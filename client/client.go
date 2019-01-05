package client

import (
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/ws-socks/ws-socks"
)

var versionCommand = &cmds.Command{
	Name:        "client",
	Summary:     "run as client mode",
	Description: "run as client program.",
	CustomFlags: false,
	HasOptions:  true,
}

func init() {
	versionCommand.Runner = &version{}
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	versionCommand.FlagSet = fs
	versionCommand.FlagSet.Usage = versionCommand.Usage // use default usage provided by cmds.Command.
	cmds.AllCommands = append(cmds.AllCommands, versionCommand)
}

type version struct{}

func (v *version) PreRun() error {
	return nil
}

func (v *version) Run() error {
	client := ws_socks.Client{
		Config: ws_socks.ClientConfig{
			LocalAddr: "localhost:1080", ServerAddr: "ws://proxy.gensh.me:10000",
		}}
	client.Start()
	return nil
}
