package wss

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

var upgrader = websocket.Upgrader{} // use default options

type WebsocksServerConfig struct {
	EnableHttp    bool
	EnableConnKey bool   // bale connection key
	ConnKey       string // connection key
}

// return a a function handling websocket requests from the peer.
func ServeWsWrapper(config WebsocksServerConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.EnableConnKey && r.Header.Get("Key") != config.ConnKey {
			w.WriteHeader(401)
			w.Write([]byte("Access denied!\n"))
			return
		}
		serveWs(w, r, config)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request, config WebsocksServerConfig) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		log.Error("Not a websocket handshake", 400)
		return
	} else if err != nil {
		http.Error(w, "Cannot setup WebSocket connection:", 400)
		log.Error("Cannot setup WebSocket connection:", err)
		return
	}
	defer ws.Close()

	// negotiate version with client.
	if err := NegVersionServer(ws); err != nil {
		return
	}

	sws := NewServerWS(ws)
	// read messages from webSocket
	for {
		_, p, err := ws.ReadMessage()
		// if WebSocket is closed by some reason, then this func will return,
		// and 'done' channel will be set, the outer func will reach to the end.
		if err != nil && err != io.EOF {
			log.Error("error reading webSocket message:", err)
			break
		}
		if err = sws.dispatchMessage(p, config); err != nil {
			log.Error("error proxy:", err)
			// break skip error
		}
	}
}
