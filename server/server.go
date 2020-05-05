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
    fs := flag.NewFlagSet("server", flag.ContinueOnError)
	serverCommand.FlagSet = fs
	serverCommand.FlagSet.StringVar(&s.address, "addr", ":1088", `listen address.`)
	serverCommand.FlagSet.BoolVar(&s.http, "http", true, `enable http and https proxy.`)
	serverCommand.FlagSet.Usage = serverCommand.Usage // use default usage provided by cmds.Command.
	serverCommand.FlagSet.BoolVar(&s.authEnable, "auth", false, `enable/disable connection authentication.`)
	serverCommand.FlagSet.StringVar(&s.authKey, "auth_key", "", "connection key for authentication. \nIf not provided, it will generate one randomly.")

	serverCommand.Runner = &s
	cmds.AllCommands = append(cmds.AllCommands, serverCommand)
}

type server struct {
	address    string
	http       bool   // enable http and https proxy
	authEnable bool   // enable authentication connection key
	authKey    string // the connection key if authentication is enabled
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
	if s.authEnable && s.authKey == "" {
		log.Trace("empty authentication key provided, now it will generate a random authentication key.")
		b, err := genRandBytes(12)
		if err != nil {
			return err
		}
		s.authKey = strings.ToUpper(hex.EncodeToString(b))
	}
	return nil
}

func (s *server) Run() error {
	config := wss.WebsocksServerConfig{EnableHttp: s.http, EnableConnKey: s.authEnable, ConnKey: s.authKey}
	http.HandleFunc("/", wss.ServeWsWrapper(config))
	if s.authEnable {
		log.Info("connection authentication key: ", s.authKey)
	}
	log.WithFields(log.Fields{
		"listen address": s.address,
	}).Info("listening for incoming messages.")

	log.Fatal(http.ListenAndServe(s.address, nil))
	return nil
}
