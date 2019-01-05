package ws_socks

import (
	"github.com/gorilla/websocket"
	"sync"
)

type ConcurrentWebSocketInterface interface {
	WSClose() error
	WriteWSJSON(data interface{}) error
}

type ConcurrentWebSocket struct {
	WsConn *websocket.Conn
	mu     sync.Mutex
}

// close websocket connection
func (wsc *ConcurrentWebSocket) WSClose() error {
	return wsc.WsConn.Close()
}

// send data to websocket
func (wsc *ConcurrentWebSocket) WriteWSJSON(data interface{}) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	return wsc.WsConn.WriteJSON(data)
}
