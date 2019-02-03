package server

import (
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/ws-socks/ws-socks"
	"github.com/genshen/ws-socks/ws-socks/ticker"
	"log"
	"net/http"
)

var serverCommand = &cmds.Command{
	Name:        "server",
	Summary:     "run as server mode",
	Description: "run as server program.",
	CustomFlags: false,
	HasOptions:  true,
}

func init() {
	var s server
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	serverCommand.FlagSet = fs
	serverCommand.FlagSet.StringVar(&s.address, "addr", ":1088", `listen address.`)
	serverCommand.FlagSet.Usage = serverCommand.Usage // use default usage provided by cmds.Command.

	serverCommand.Runner = &s
	cmds.AllCommands = append(cmds.AllCommands, serverCommand)
}

type server struct {
	address string
}

func (v *server) PreRun() error {
	return nil
}

func (v *server) Run() error {
	// new time ticker to flush data into websocket (to client).
	tick := ticker.NewTicker()
	tick.Start()
	defer tick.Stop()

	http.HandleFunc("/", ws_socks.ServeWs)
	log.Println("listening on ", v.address)
	log.Fatal(http.ListenAndServe(v.address, nil))
	return nil
}
