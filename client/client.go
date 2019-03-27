package client

import (
	"errors"
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
	"github.com/genshen/wssocks/wss/ticker"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

var clientCommand = &cmds.Command{
	Name:        "client",
	Summary:     "run as client mode",
	Description: "run as client program.",
	CustomFlags: false,
	HasOptions:  true,
}

func init() {
	var client client
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	clientCommand.FlagSet = fs
	clientCommand.FlagSet.StringVar(&client.address, "addr", ":1080", `listen address of socks5.`)
	clientCommand.FlagSet.StringVar(&client.remote, "remote", "", `server address and port(e.g: ws://example.com:1088).`)
	clientCommand.FlagSet.StringVar(&client.key, "key", "", `connection key.`)
	clientCommand.FlagSet.IntVar(&client.ticker, "ticker", 0, `ticker(ms) to send data to client.`)

	clientCommand.FlagSet.Usage = clientCommand.Usage // use default usage provided by cmds.Command.
	clientCommand.Runner = &client

	cmds.AllCommands = append(cmds.AllCommands, clientCommand)
}

type client struct {
	address      string
	remote       string
	ticker       int
	remoteUrl    *url.URL
	remoteHeader http.Header // header in websocket request(default is nil)
	key string
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
	c.remoteHeader = make(http.Header)
	if c.key != "" {
		c.remoteHeader.Set("Key", c.key)
	}
	// loading and execute plugin
	if clientPlugin.HasPlugin() {
		// in the plugin, we may add http header and modify remote address.
		clientPlugin.RedirectPlugin.BeforeRequest(c.remoteUrl, c.remoteHeader)
	}
	return nil
}

func (c *client) Run() error {
	// start websocket connection (to remote server).
	log.Println("connecting to ", c.remoteUrl.String())
	wsc, err := wss.NewWebSocketClient(c.remoteUrl.String(), c.remoteHeader)
	if err != nil {
		log.Fatal("establishing connection error:", err)
	}
	log.Println("connected to ", c.remoteUrl.String())
	// todo chan for wsc and tcp accept
	defer wsc.WSClose()

	// negotiate version
	if version, err := wss.NegVersionClient(wsc.WsConn); err != nil {
		log.Println("server version {version code:", version.VersionCode,
			", version number:", version.Version,
			", update address:", version.UpdateAddr, "}")
		return err
	}

	// start websocket message listen.
	go func() {
		if err := wsc.ListenIncomeMsg(); err != nil {
			log.Println("error websocket read:", err)
		}
	}()
	// send heart beats.
	go func() {
		if err := wsc.HeartBeat(); err != nil {
			log.Println("heartbeat ending", err)
		}
	}()

	// new time ticker to flush data into websocket (server).
	var tick *ticker.Ticker = nil
	if c.ticker != 0 {
		tick = ticker.NewTicker()
		tick.Start(time.Microsecond * time.Duration(100))
		defer tick.Stop()
	}

	// start listen for socks5 connection.
	s, err := net.Listen("tcp", c.address)
	if err != nil {
		log.Panic(err)
	}
	var client wss.Client
	for {
		log.Println("size of connector:", wsc.ConnSize())
		c, err := s.Accept()
		if err != nil {
			log.Panic(err)
			break
		}
		go func() {
			err := client.Reply(c, func(conn *net.TCPConn, addr string) error {
				proxy := wsc.NewProxy(conn)
				proxy.Serve(wsc, tick, addr)
				wsc.TellClose(proxy.Id)
				return nil // todo error
			})
			if err != nil {
				log.Println(err)
			}
		}()
	}
	return nil
}
