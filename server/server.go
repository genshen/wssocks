package server

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"github.com/genshen/cmds"
	"github.com/genshen/wssocks/wss"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
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
	serverCommand.FlagSet.BoolVar(&s.keyEnable, "key", false, `enable/disable connection key.`)

	serverCommand.Runner = &s
	cmds.AllCommands = append(cmds.AllCommands, serverCommand)
}

type server struct {
	address   string
	http      bool   // enable http and https proxy
	keyEnable bool   // enable connection key
	key       string // the connection key if enabled
}

func genRandBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *server) PreRun() error {
	if s.keyEnable {
		b, err := genRandBytes(12)
		if err != nil {
			return err
		}
		s.key = strings.ToUpper(hex.EncodeToString(b))
	}
	return nil
}

func (s *server) Run() error {
	config := wss.WebsocksServerConfig{EnableHttp: s.http, EnableConnKey: s.keyEnable, ConnKey: s.key}
	http.HandleFunc("/", wss.ServeWsWrapper(config))
	if s.keyEnable {
		log.Info("connection secret key: ", s.key)
	}
	log.WithFields(log.Fields{
		"listen address": s.address,
	}).Info("listening for incoming messages.")

	log.Fatal(http.ListenAndServe(s.address, nil))
	return nil
}
