package server

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
    "net/http"
    "strings"

	"github.com/genshen/cmds"
    _ "github.com/genshen/wssocks/server/statik"
	"github.com/genshen/wssocks/wss"
	"github.com/genshen/wssocks/wss/status"
    "github.com/rakyll/statik/fs"
	log "github.com/sirupsen/logrus"
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
	serverCommand.FlagSet.StringVar(&s.wsBasePath, "ws_base_path", "/", "base path for serving websocket.")
	serverCommand.FlagSet.BoolVar(&s.http, "http", true, `enable http and https proxy.`)
	serverCommand.FlagSet.BoolVar(&s.authEnable, "auth", false, `enable/disable connection authentication.`)
	serverCommand.FlagSet.StringVar(&s.authKey, "auth_key", "", "connection key for authentication. \nIf not provided, it will generate one randomly.")
    serverCommand.FlagSet.BoolVar(&s.status, "status", false, `enable/disable serving status page.`)
	serverCommand.FlagSet.Usage = serverCommand.Usage // use default usage provided by cmds.Command.

	serverCommand.Runner = &s
	cmds.AllCommands = append(cmds.AllCommands, serverCommand)
}

type server struct {
	address    string
	wsBasePath string // base path for serving websocket and status page
	http       bool   // enable http and https proxy
	authEnable bool   // enable authentication connection key
	authKey    string // the connection key if authentication is enabled
    status     bool   // enable service status page
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
	// set base url
	if s.wsBasePath == "" {
		s.wsBasePath = "/"
	}
	// complete prefix and suffix
	if !strings.HasPrefix(s.wsBasePath, "/") {
		s.wsBasePath = "/" + s.wsBasePath
	}
	if !strings.HasSuffix(s.wsBasePath, "/") {
		s.wsBasePath = s.wsBasePath + "/"
	}
	return nil
}

func (s *server) Run() error {
    config := wss.WebsocksServerConfig{EnableHttp: s.http, EnableConnKey: s.authEnable, ConnKey: s.authKey, EnableStatusPage: s.status}
    hc := wss.NewHubCollection()

    http.Handle(s.wsBasePath, wss.NewServeWS(hc,config))
    if s.status {
        statikFS, err := fs.New()
        if err != nil {
            log.Fatal(err)
        }
        http.Handle("/status/", http.StripPrefix("/status", http.FileServer(statikFS)))
        http.Handle("/api/status/", status.NewStatusHandle(hc, s.http, s.authEnable, s.wsBasePath))
    }

	if s.authEnable {
		log.Info("connection authentication key: ", s.authKey)
	}
    if s.status {
        log.Info("service status page is enabled at `/status` endpoint")
    }

    listenAddrToLog := s.address + s.wsBasePath
    if s.wsBasePath == "/"{
		listenAddrToLog = s.address
	}
	log.WithFields(log.Fields{
		"listen address": listenAddrToLog,
	}).Info("listening for incoming messages.")

	log.Fatal(http.ListenAndServe(s.address, nil))
	return nil
}
