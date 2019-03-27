package main

import (
	"github.com/genshen/cmds"
	_ "github.com/DefinitlyEvil/wssocks/client"
	_ "github.com/DefinitlyEvil/wssocks/server"
	_ "github.com/DefinitlyEvil/wssocks/version"
	"log"
)

func main() {
	cmds.SetProgramName("wssocks")
	if err := cmds.Parse(); err != nil {
		log.Fatal(err)
	}
}
