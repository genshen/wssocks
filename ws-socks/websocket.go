package ws_socks

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"time"
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

	var wsBuff WebSocketBufferWriter
	defer wsBuff.Flush(websocket.BinaryMessage, ws)

	done := make(chan bool, 2)
	setDone := func() { done <- true }
	stopper := make(chan bool) // timer stopper
	// check webSocketWriterBuffer(if not empty,then write back to webSocket) every 120 ms.
	writeBufferToWebSocket := func() {
		defer setDone()
		tick := time.NewTicker(time.Millisecond * time.Duration(10))
		//for range time.Tick(120 * time.Millisecond){}
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if err := wsBuff.Flush(websocket.BinaryMessage, ws); err != nil {
					log.Println("Error: error sending data via webSocket:", err)
					return
				}
			case <-stopper:
				return
			}
		}
	}

	dispatchWebsocket := func() { // read messages from webSocket
		defer setDone()
		//	for {
		_, p, err := ws.ReadMessage()
		// if WebSocket is closed by some reason, then this func will return,
		// and 'done' channel will be set, the outer func will reach to the end.
		// then ssh session will be closed in defer.
		if err != nil {
			log.Println("Error: error reading webSocket message:", err)
			return
		}
		if err = dispatchMessage(ws, &wsBuff, p); err != nil {
			log.Println("Error: error proxy:", err)
			return
		}
		// }
	}

	go dispatchWebsocket()
	go writeBufferToWebSocket()

	<-done
	stopper <- true
}

// in this case, one ws only handle one proxy.
func dispatchMessage(conn *websocket.Conn, buff *WebSocketBufferWriter, data []byte) error {
	var socketData json.RawMessage
	socketStream := WebSocketMessage{
		Data: &socketData,
	}
	if err := json.Unmarshal(data, &socketStream); err != nil {
		return nil // skip error
	}
	var proxyMsg ProxyMessage
	if err := json.Unmarshal(socketData, &proxyMsg); err != nil {
		return nil
	}

	log.Println("info", "proxy to:", proxyMsg.Addr)
	server, err := net.DialTimeout("tcp", proxyMsg.Addr, time.Second*8) // todo config timeout
	if err != nil {
		log.Println(err)
		return err
	}

	defer server.Close()
	if _, err := buff.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		return err
	}
	log.Println("info", "connected to:", proxyMsg.Addr)
	go io.Copy(buff, server)

	// todo use go func
	for {
		if _, requestWsData, err := conn.ReadMessage(); err != nil {
			return errors.New("error read ws data from client," + err.Error())
		} else {
			log.Println("parsing data from client", string(requestWsData))
			// parse data
			var socketDataRaw json.RawMessage
			dataStream := WebSocketMessage{
				Data: &socketDataRaw,
			}
			if err := json.Unmarshal(requestWsData, &dataStream); err != nil {
				log.Println("parsing request error,", err)
				return nil // skip error
			}
			var requestMsg RequestMessage
			if err := json.Unmarshal(socketDataRaw, &requestMsg); err != nil {
				log.Println("parsing request data error,", err)
				return nil
			}

			// copy data
			if decodeBytes, err := base64.StdEncoding.DecodeString(requestMsg.DataBase64); err != nil { // todo ignore error
				log.Println("bash64 decode error,", err)
				return nil // skip error
			} else {
				if _, err := server.Write(decodeBytes); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
