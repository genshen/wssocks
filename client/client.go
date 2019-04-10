package client

import (
	"errors"
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
	"github.com/genshen/wssocks/wss/ticker"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
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
	fs := flag.NewFlagSet(CommandNameClient, flag.ExitOnError)
	clientCommand.FlagSet = fs
	clientCommand.FlagSet.StringVar(&client.address, "addr", ":1080", `listen address of socks5.`)
	clientCommand.FlagSet.StringVar(&client.remote, "remote", "", `server address and port(e.g: ws://example.com:1088).`)
	clientCommand.FlagSet.IntVar(&client.ticker, "ticker", 0, `ticker(ms) to send data to client.`)

	clientCommand.FlagSet.Usage = clientCommand.Usage // use default usage provided by cmds.Command.
	clientCommand.Runner = &client

	cmds.AllCommands = append(cmds.AllCommands, clientCommand)
}

type client struct {
	address   string
	remote    string
	ticker    int
	remoteUrl *url.URL
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

	return nil
}

func (c *client) Run() error {
	// start websocket connection (to remote server).
	log.Println("connecting to ", c.remoteUrl.String())

	dialer := websocket.DefaultDialer
	wsHeader := make(http.Header) // header in websocket request(default is nil)

	// loading and execute plugin
	if clientPlugin.HasPlugin() {
		// in the plugin, we may add http header/dialer and modify remote address.
		if err := clientPlugin.RedirectPlugin.BeforeRequest(dialer, c.remoteUrl, wsHeader); err != nil {
			return err
		}
	}

	wsc, err := wss.NewWebSocketClient(websocket.DefaultDialer, c.remoteUrl.String(), wsHeader)
	if err != nil {
		log.Fatal("establishing connection error:", err)
	}
	log.Println("connected to ", c.remoteUrl.String())
	// todo chan for wsc and tcp accept
	defer wsc.WSClose()

	// negotiate version
	if version, err := wss.ExchangeVersion(wsc.WsConn); err != nil {
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
		tick.Start(time.Microsecond * time.Duration(c.ticker))
		defer tick.Stop()
	}

	// start listen for socks5 connection.
	if err := wss.ListenAndServe(wsc, tick, c.address); err != nil {
		return err
	}
	return nil
}
