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
	serverCommand.FlagSet.BoolVar(&s.http, "http", true, `enable http and https proxy.`)
	serverCommand.FlagSet.Usage = serverCommand.Usage // use default usage provided by cmds.Command.

	serverCommand.Runner = &s
	cmds.AllCommands = append(cmds.AllCommands, serverCommand)
}

type server struct {
	address string
	http    bool // enable http and https proxy
}

func (s *server) PreRun() error {
	return nil
}

func (s *server) Run() error {
	config := wss.WebsocksServerConfig{EnableHttp: s.http}
	http.HandleFunc("/", wss.ServeWsWrapper(config))
	log.WithFields(log.Fields{
		"listen address": s.address,
	}).Info("listening for incoming messages.")
	log.Fatal(http.ListenAndServe(s.address, nil))
	return nil
}
