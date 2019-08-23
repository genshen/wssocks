package server

import (
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
	log "github.com/sirupsen/logrus"
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

func (s *server) PreRun() error {
	return nil
}

func (s *server) Run() error {
	// new time ticker to flush data into websocket (to client).
	http.HandleFunc("/", wss.ServeWs)
	log.WithFields(log.Fields{
		"listen address": s.address,
	}).Info("listening on income message.")
	log.Fatal(http.ListenAndServe(s.address, nil))
	return nil
}
