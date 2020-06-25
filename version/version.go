package version

import (
	"flag"
	"fmt"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
)

const VERSION = wss.CoreVersion
var buildHash = "none"
var buildTime = "none"

var versionCommand = &cmds.Command{
	Name:        "version",
	Summary:     "show version",
	Description: "print current version.",
	CustomFlags: false,
	HasOptions:  false,
}

func init() {
	versionCommand.Runner = &version{}
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	versionCommand.FlagSet = fs
	versionCommand.FlagSet.Usage = versionCommand.Usage // use default usage provided by cmds.Command.
	cmds.AllCommands = append(cmds.AllCommands, versionCommand)
}

type version struct{}

func (v *version) PreRun() error {
	return nil
}

func (v *version) Run() error {
	fmt.Printf("version: %s.\n", VERSION)
	fmt.Printf("protocol version: %d\n", wss.VersionCode)
	fmt.Printf("commit: %s\n", buildHash)
	fmt.Printf("build time: %s\n", buildTime)
	fmt.Println("Author: genshen<genshenchu@gmail.com>")
	fmt.Println("github: https://github.com/genshen/wssocks")
	return nil
}
