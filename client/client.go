package client

import (
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/ws-socks/ws-socks"
	"github.com/genshen/ws-socks/ws-socks/ticker"
	"log"
	"net"
)

var clientCommand = &cmds.Command{
	Name:        "client",
	Summary:     "run as client mode",
	Description: "run as client program.",
	CustomFlags: false,
	HasOptions:  true,
}

func init() {
	clientCommand.Runner = &client{}
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	clientCommand.FlagSet = fs
	clientCommand.FlagSet.Usage = clientCommand.Usage // use default usage provided by cmds.Command.
	cmds.AllCommands = append(cmds.AllCommands, clientCommand)
}

type client struct{}

func (v *client) PreRun() error {
	return nil
}

func (v *client) Run() error {
	client := ws_socks.Client{
		Config: ws_socks.ClientConfig{
			LocalAddr: "localhost:1080", ServerAddr: "ws://proxy.gensh.me:10000",
		}}

	// start websocket connection (to remote server).
	wsc := ws_socks.WebSocketClient{}
	wsc.Connect(client.Config.ServerAddr)
	log.Println("connected")
	// todo chan for wsc and tcp accept
	defer wsc.WSClose()
	// start websocket message listen.
	go wsc.ListenIncomeMsg()

	// new time ticker to flush data into websocket (server).
	tick := ticker.NewTicker()
	tick.Start()
	defer tick.Stop()

	// start listen for socks5 connection.
	s, err := net.Listen("tcp", client.Config.LocalAddr)
	if err != nil {
		log.Panic(err)
	}
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
				proxy.Serve(&wsc, tick, addr)
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
