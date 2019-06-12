package server

import (
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
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
	serverCommand.FlagSet.IntVar(&s.ticker, "ticker", 0, `ticker(ms) to send data to client.`)
	serverCommand.FlagSet.Usage = serverCommand.Usage // use default usage provided by cmds.Command.

	serverCommand.Runner = &s
	cmds.AllCommands = append(cmds.AllCommands, serverCommand)
}

type server struct {
	address string
	ticker  int
}

func (s *server) PreRun() error {
	return nil
}

func (s *server) Run() error {
	if s.ticker != 0 {
		ticker := wss.StartTicker(time.Microsecond * time.Duration(100))
		defer ticker.Stop()
	}

	// new time ticker to flush data into websocket (to client).
	http.HandleFunc("/", wss.ServeWs)
	log.WithFields(log.Fields{
		"listen address": s.address,
	}).Info("listening on income message.")
	log.Fatal(http.ListenAndServe(s.address, nil))
	return nil
}
