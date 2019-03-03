package main

import (
	"github.com/genshen/cmds"
	_ "github.com/genshen/wssocks/client"
	_ "github.com/genshen/wssocks/server"
	_ "github.com/genshen/wssocks/version"
	"log"
)

func main() {
	cmds.SetProgramName("wssocks")
	if err := cmds.Parse(); err != nil {
		log.Fatal(err)
	}
}
