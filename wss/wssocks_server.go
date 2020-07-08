package wss

import (
    "context"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
    "nhooyr.io/websocket"
)

type WebsocksServerConfig struct {
	EnableHttp    bool
	EnableConnKey bool   // bale connection key
	ConnKey       string // connection key
}

// return a a function handling websocket requests from the peer.
func ServeWsWrapper(hc *HubCollection, config WebsocksServerConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.EnableConnKey && r.Header.Get("Key") != config.ConnKey {
			w.WriteHeader(401)
			w.Write([]byte("Access denied!\n"))
			return
		}
		serveWs(w, r, hc, config)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request, hc *HubCollection, config WebsocksServerConfig) {
    wc, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error(err)
	}
    defer wc.Close(websocket.StatusNormalClosure, "the sky is falling")

    ctx, cancel := context.WithCancel(r.Context())
    defer cancel()

	// negotiate version with client.
    if err := NegVersionServer(ctx, wc); err != nil {
		return
	}

	hub := hc.AddHub(wc)
	defer hc.RemoveProxy(hub.id)
	defer hub.Close()
    go hub.Run()
	// read messages from webSocket
	for {
        msgType, p, err := wc.Read(ctx) // fixme context
		// if WebSocket is closed by some reason, then this func will return,
		// and 'done' channel will be set, the outer func will reach to the end.
		if err != nil && err != io.EOF {
			log.Error("error reading webSocket message:", err)
			break
		}
        if err = dispatchMessage(hub, msgType, p, config); err != nil {
			log.Error("error proxy:", err)
			// break skip error
		}
	}
}
