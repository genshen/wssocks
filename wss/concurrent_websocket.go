package wss

import (
	"context"
	"encoding/base64"
	"github.com/segmentio/ksuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type ConcurrentWebSocketInterface interface {
	WSClose() error
	WriteWSJSON(data interface{}) error
}

// add lock to websocket connection to make sure only one goroutine can write this websocket.
type ConcurrentWebSocket struct {
	WsConn *websocket.Conn
}

// close websocket connection
func (wsc *ConcurrentWebSocket) WSClose() error {
	return wsc.WsConn.Close(websocket.StatusNormalClosure, "")
}

// write message to websocket, the data is fixed format @ProxyData
// id: connection id
// data: data to be written
func (wsc *ConcurrentWebSocket) WriteProxyMessage(ctx context.Context, id ksuid.KSUID, tag int, data []byte) error {
	dataBase64 := base64.StdEncoding.EncodeToString(data)
	jsonData := WebSocketMessage{
		Id:   id.String(),
		Type: WsTpData,
		Data: ProxyData{Tag: tag, DataBase64: dataBase64},
	}
	return wsjson.Write(ctx, wsc.WsConn, &jsonData)
}

type WebSocketWriter struct {
	WSC  *ConcurrentWebSocket
	Id   ksuid.KSUID // connection id.
	Ctx  context.Context
	Type int // type of trans data.
}

func (writer *WebSocketWriter) Write(buffer []byte) (n int, err error) {
	if err := writer.WSC.WriteProxyMessage(writer.Ctx, writer.Id, TagData, buffer); err != nil {
		return 0, err
	} else {
		return len(buffer), nil
	}
}
