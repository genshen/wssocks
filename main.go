package main

import (
	"github.com/genshen/cmds"
	_ "github.com/genshen/ws-socks/client"
	_ "github.com/genshen/ws-socks/server"
	_ "github.com/genshen/ws-socks/version"
	"log"
)

func main() {
	cmds.SetProgramName("wssocks")
	if err := cmds.Parse(); err != nil {
		log.Fatal(err)
	}
}
