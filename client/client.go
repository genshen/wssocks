package client

import (
	"context"
	"errors"
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
	"github.com/genshen/wssocks/wss/term_view"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
    "time"
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
    fs := flag.NewFlagSet(CommandNameClient, flag.ContinueOnError)
	clientCommand.FlagSet = fs
	clientCommand.FlagSet.StringVar(&client.address, "addr", ":1080", `listen address of socks5 proxy.`)
	clientCommand.FlagSet.BoolVar(&client.http, "http", false, `enable http and https proxy.`)
	clientCommand.FlagSet.StringVar(&client.httpAddr, "http-addr", ":1086", `listen address of http proxy (if enabled).`)
	clientCommand.FlagSet.StringVar(&client.remote, "remote", "", `server address and port(e.g: ws://example.com:1088).`)
	clientCommand.FlagSet.StringVar(&client.key, "key", "", `connection key.`)

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
	key string
}

type Handles struct {
	wsc        *wss.WebSocketClient
	hb         *wss.HeartBeat
	httpServer *http.Server
	cl         *wss.Client
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

	wsHeader := make(http.Header) // header in websocket request(default is nil)

	if c.key != "" {
		wsHeader.Set("Key", c.key)
	}

	httpClient := http.Client{}
	// loading and execute plugin
	if clientPlugin.HasRedirectPlugin() {
		// in the plugin, we may add http header/dialer and modify remote address.
        if err := clientPlugin.RedirectPlugin.BeforeRequest(&httpClient, c.remoteUrl, &wsHeader); err != nil {
			return err
		}
	}

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute) // fixme
    defer cancel()
    wsc, err := wss.NewWebSocketClient(ctx, c.remoteUrl.String(), &httpClient, wsHeader)
	if err != nil {
		log.Fatal("establishing connection error:", err)
	}
	log.WithFields(log.Fields{
		"remote": c.remoteUrl.String(),
	}).Info("connected to wssocks server.")
	// todo chan for wsc and tcp accept
	defer wsc.WSClose()

	// negotiate version
    if version, err := wss.ExchangeVersion(ctx, wsc.WsConn); err != nil {
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

	var hdl Handles
	hdl.wsc = wsc

	var wg sync.WaitGroup
	var once sync.Once // wait for one of go func
	wg.Add(3)          // wait for all go func

	// stop all connections or tasks, if one of tasks is finished.
	closeAll := func() {
		if hdl.cl != nil {
			hdl.cl.Close(false)
		}
		if hdl.httpServer != nil {
			hdl.httpServer.Shutdown(context.TODO())
		}
		if hdl.hb != nil {
			hdl.hb.Close()
		}
		if hdl.wsc != nil {
			hdl.wsc.Close()
		}
	}

	// start websocket message listen.
	go func() {
		defer wg.Done()
        defer once.Do(closeAll)
        if err := wsc.ListenIncomeMsg(1 << 29); err != nil {
			log.Error("error websocket read:", err)
		}
	}()
	// send heart beats.
    heartbeat, hbCtx := wss.NewHeartBeat(wsc)
    hdl.hb = heartbeat
	go func() {
		defer wg.Done()
        defer once.Do(closeAll)
        if err := hdl.hb.Start(hbCtx); err != nil {
			log.Info("heartbeat ending", err)
		}
	}()

	record := wss.NewConnRecord()
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		// if it is tty, use term_view as output, and set onChange function to update output
		plog := term_view.NewPLog(record)
		log.SetOutput(plog) // change log stdout to plog

        record.OnChange = func(wss.ConnStatus) {
			// update log
			plog.SetLogBuffer(record) // call Writer.Write() to set log data into buffer
			plog.Writer.Flush(nil)    // flush buffer
		}
    } else {
        record.OnChange = func(status wss.ConnStatus) {
            if status.IsNew {
                log.WithField("address", status.Address).Traceln("new proxy connection")
            } else {
                log.WithField("address", status.Address).Traceln("close proxy connection")
            }
        }
	}

	// http listening
	if c.http {
		wg.Add(1)
		log.WithField("http listen address", c.httpAddr).
			Info("listening on local address for incoming proxy requests.")
		go func() {
			defer wg.Done()
            defer once.Do(closeAll)
			handle := wss.NewHttpProxy(wsc, record)
			hdl.httpServer = &http.Server{Addr: c.httpAddr, Handler: &handle}
			if err := hdl.httpServer.ListenAndServe(); err != nil {
				log.Errorln(err)
			}
		}()
	}

	// start listen for socks5 and https connection.
	hdl.cl = wss.NewClient()
	go func() {
        defer wg.Done()
        defer once.Do(closeAll)
		if err := hdl.cl.ListenAndServe(record, wsc, c.address, c.http, func() {
			if c.http {
				log.WithField("socks5 listen address", c.address).
					WithField("https listen address", c.address).
					Info("listening on local address for incoming proxy requests.")
			} else {
				log.WithField("socks5 listen address", c.address).
					Info("listening on local address for incoming proxy requests.")
			}
		}); err != nil {
			log.Errorln(err)
		}
	}()

	go func() {
		firstInterrupt := true
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		for { // accept multiple signal
			select {
			case <-c:
				if firstInterrupt {
					log.Println("press CTRL+C to force exit")
					firstInterrupt = false
					go func() {
						// stop tasks in signal
						once.Do(func() {
							if hdl.cl != nil {
								hdl.cl.Close(true)
							}
							if hdl.httpServer != nil {
								hdl.httpServer.Shutdown(context.TODO())
							}
							if hdl.hb != nil {
								hdl.hb.Close()
							}
							if hdl.wsc != nil {
								hdl.wsc.Close()
							}
						})
					}()
				} else {
					os.Exit(0)
				}
			}
		}
	}()

	wg.Wait() // wait all tasks finished
	// about exit: 1. press ctrl+c, it will wait active connection to finish.
	// 2. press twice, force exit.
	// 3. one of tasks error, exit immediately.
	// 4. close server, then client exit (the same as 3).
	return nil
}
