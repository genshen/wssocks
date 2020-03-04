package version

import (
	"flag"
	"fmt"
	"github.com/genshen/cmds"
	"github.com/DefinitlyEvil/wssocks/wss"
)

const VERSION = wss.CoreVersion

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
	fmt.Printf("version\t %s.\n", VERSION)
	fmt.Printf("protocol version\t %d\n", wss.VersionCode)
	fmt.Println("Author\t genshen<genshenchu@gmail.com>")
	fmt.Println("github \t https://github.com/DefinitlyEvil/wssocks")
	return nil
}
