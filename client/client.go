package client

import (
	"errors"
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
	"github.com/genshen/wssocks/wss/term_view"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"sync"
)

const CommandNameClient = "client"

var clientCommand = &cmds.Command{
	Name:        CommandNameClient,
	Summary:     "run as client mode",
	Description: "run as client program.",
	CustomFlags: false,
	HasOptions:  true,
}

func init() {
	var client client
	fs := flag.NewFlagSet(CommandNameClient, flag.ExitOnError)
	clientCommand.FlagSet = fs
	clientCommand.FlagSet.StringVar(&client.address, "addr", ":1080", `listen address of socks5 proxy.`)
	clientCommand.FlagSet.BoolVar(&client.http, "http", false, `enable http and https proxy.`)
	clientCommand.FlagSet.StringVar(&client.httpAddr, "http-addr", ":1086", `listen address of http proxy (if enabled).`)
	clientCommand.FlagSet.StringVar(&client.remote, "remote", "", `server address and port(e.g: ws://example.com:1088).`)

	clientCommand.FlagSet.Usage = clientCommand.Usage // use default usage provided by cmds.Command.
	clientCommand.Runner = &client

	cmds.AllCommands = append(cmds.AllCommands, clientCommand)
}

type client struct {
	address   string   // local listening address
	http      bool     // enable http and https proxy
	httpAddr  string   // listen address of http and https(if it is enabled)
	remote    string   // string usr of server
	remoteUrl *url.URL // url of server
	//	remoteHeader http.Header
}

func (c *client) PreRun() error {
	// check remote address
	if c.remote == "" {
		return errors.New("empty remote address")
	}
	if u, err := url.Parse(c.remote); err != nil {
		return err
	} else {
		c.remoteUrl = u
	}

	if c.http {
		log.Info("http(s) proxy is enabled.")
	} else {
		log.Info("http(s) proxy is disabled.")
	}
	return nil
}

func (c *client) Run() error {
	// start websocket connection (to remote server).
	log.WithFields(log.Fields{
		"remote": c.remoteUrl.String(),
	}).Info("connecting to wssocks server.")

	dialer := websocket.DefaultDialer
	wsHeader := make(http.Header) // header in websocket request(default is nil)

	// loading and execute plugin
	if clientPlugin.HasRedirectPlugin() {
		// in the plugin, we may add http header/dialer and modify remote address.
		if err := clientPlugin.RedirectPlugin.BeforeRequest(dialer, c.remoteUrl, wsHeader); err != nil {
			return err
		}
	}

	wsc, err := wss.NewWebSocketClient(websocket.DefaultDialer, c.remoteUrl.String(), wsHeader)
	if err != nil {
		log.Fatal("establishing connection error:", err)
	}
	log.WithFields(log.Fields{
		"remote": c.remoteUrl.String(),
	}).Info("connected to wssocks server.")
	// todo chan for wsc and tcp accept
	defer wsc.WSClose()

	// negotiate version
	if version, err := wss.ExchangeVersion(wsc.WsConn); err != nil {
		return err
	} else {
		if clientPlugin.HasVersionPlugin() {
			if err := clientPlugin.VersionPlugin.OnServerVersion(version); err != nil {
				return err
			}
		} else {
			log.WithFields(log.Fields{
				"compatible version code": version.CompVersion,
				"version code":            version.VersionCode,
				"version number":          version.Version,
			}).Info("server version")

			if version.CompVersion > wss.VersionCode || wss.VersionCode > version.VersionCode {
				return errors.New("incompatible protocol version of client and server")
			}
			if version.Version != wss.CoreVersion {
				log.WithFields(log.Fields{
					"client wssocks version": wss.CoreVersion,
					"server wssocks version": version.Version,
				}).Warning("different version of client and server wssocks")
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(3) // 3 go func
	// start websocket message listen.
	go func() {
		defer wg.Done()
		if err := wsc.ListenIncomeMsg(); err != nil {
			log.Error("error websocket read:", err)
		}
	}()
	// send heart beats.
	hb := wss.NewHeartBeat(wsc)
	go func() {
		defer wg.Done()
		if err := hb.Start(); err != nil {
			log.Info("heartbeat ending", err)
		}
	}()

	record := wss.NewConnRecord()
	plog := term_view.NewPLog(record)
	log.SetOutput(plog) // change log stdout to plog

	record.OnChange = func() {
		// update log
		plog.SetLogBuffer(record) // call Writer.Write() to set log data into buffer
		plog.Writer.Flush(nil)    // flush buffer
	}

	// http listening
	if c.http {
		wg.Add(1)
		log.WithField("http listen address", c.httpAddr).
			Info("listening on local address for incoming proxy requests.")
		go func() {
			defer wg.Done()
			handle := wss.NewHttpProxy(wsc, record)
			server := http.Server{Addr: c.httpAddr, Handler: &handle}
			if err := server.ListenAndServe(); err != nil {
				log.Fatalln(err)
			}
		}()
	}

	// start listen for socks5 and https connection.
	cl := wss.NewClient()
	go func() {
		defer wg.Done()
		if err := cl.ListenAndServe(record, wsc, c.address, c.http, func() {
			if c.http {
				log.WithField("socks5 listen address", c.address).
					WithField("https listen address", c.address).
					Info("listening on local address for incoming proxy requests.")
			} else {
				log.WithField("socks5 listen address", c.address).
					Info("listening on local address for incoming proxy requests.")
			}
		}); err != nil {
			log.Fatalln(err)
		}
	}()

	wg.Wait()
	return nil
}
