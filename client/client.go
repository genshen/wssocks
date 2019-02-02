package client

import (
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/ws-socks/ws-socks"
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
	client.Start()
	return nil
}
