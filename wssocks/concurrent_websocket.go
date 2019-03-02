package ws_socks

import (
	"encoding/base64"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
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

// write message to websocket, the data is fixed format @ProxyData
// id: connection id
// data: data to be written
func (wsc *ConcurrentWebSocket) WriteMessage(id ksuid.KSUID, data []byte) error {
	dataBase64 := base64.StdEncoding.EncodeToString(data)
	jsonData := WebSocketMessage{
		Id:   id.String(),
		Type: WsTpData,
		Data: ProxyData{DataBase64: dataBase64},
	}
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	return wsc.WsConn.WriteJSON(jsonData)
}

// send data to websocket
func (wsc *ConcurrentWebSocket) WriteWSJSON(data interface{}) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	return wsc.WsConn.WriteJSON(data)
}
