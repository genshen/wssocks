package wss

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

// add lock to websocket connection to make sure only one goroutine can write this websocket.
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
func (wsc *ConcurrentWebSocket) WriteProxyMessage(id ksuid.KSUID, tag int, data []byte) error {
	dataBase64 := base64.StdEncoding.EncodeToString(data)
	jsonData := WebSocketMessage{
		Id:   id.String(),
		Type: WsTpData,
		Data: ProxyData{Tag: tag, DataBase64: dataBase64},
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

type WebSocketWriter struct {
	WSC  *ConcurrentWebSocket
	Id   ksuid.KSUID // connection id.
	Type int         // type of trans data.
}

func (writer *WebSocketWriter) Write(buffer []byte) (n int, err error) {
	if err := writer.WSC.WriteProxyMessage(writer.Id, TagData, buffer); err != nil {
		return 0, err
	} else {
		return len(buffer), nil
	}
}
