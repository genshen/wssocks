package wss

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{} // use default options

// listen http port and serve it
// serveWs handles websocket requests from the peer.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		log.Println("Error: Not a websocket handshake", 400)
		return
	} else if err != nil {
		http.Error(w, "Cannot setup WebSocket connection:", 400)
		log.Println("Error: Cannot setup WebSocket connection:", err)
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
		log.Println("conn size: ", sws.GetConnectorSize())
		_, p, err := ws.ReadMessage()
		// if WebSocket is closed by some reason, then this func will return,
		// and 'done' channel will be set, the outer func will reach to the end.
		// then ssh session will be closed in defer.
		if err != nil {
			log.Println("Error: error reading webSocket message:", err)
			break
		}
		if err = sws.dispatchMessage(p); err != nil {
			log.Println("Error: error proxy:", err)
			// break skip error
		}
	}
}
