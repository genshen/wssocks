package wss

import (
	"context"
	"encoding/base64"
	"sync"

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

type webSocketWriter struct {
	WSC  *ConcurrentWebSocket
	Id   ksuid.KSUID // connection id.
	Ctx  context.Context
	Type int // type of trans data.
	Mu   *sync.Mutex
}

func NewWebSocketWriter(wsc *ConcurrentWebSocket, id ksuid.KSUID, ctx context.Context) *webSocketWriter {
	return &webSocketWriter{WSC: wsc, Id: id, Ctx: ctx}
}

func NewWebSocketWriterWithMutex(wsc *ConcurrentWebSocket, id ksuid.KSUID, ctx context.Context) *webSocketWriter {
	return &webSocketWriter{WSC: wsc, Id: id, Ctx: ctx, Mu: &sync.Mutex{}}
}

func (writer *webSocketWriter) CloseWsWriter(cancel context.CancelFunc) {
	if writer.Mu != nil {
		writer.Mu.Lock()
		defer writer.Mu.Unlock()
	}
	cancel()
}

func (writer *webSocketWriter) Write(buffer []byte) (n int, err error) {
	if writer.Mu != nil {
		writer.Mu.Lock()
		defer writer.Mu.Unlock()
	}
	// make sure context is not Canceled/DeadlineExceeded before Write.
	if writer.Ctx.Err() != nil {
		return 0, writer.Ctx.Err()
	}
	if err := writer.WSC.WriteProxyMessage(writer.Ctx, writer.Id, TagData, buffer); err != nil {
		return 0, err
	} else {
		return len(buffer), nil
	}
}

// 连接关闭
func (writer *webSocketWriter) WriteEOF() {
	if writer.Mu != nil {
		writer.Mu.Lock()
		defer writer.Mu.Unlock()
	}
	// make sure context is not Canceled/DeadlineExceeded before Write.
	if writer.Ctx.Err() != nil {
		return
	}
	writer.WSC.WriteProxyMessage(writer.Ctx, writer.Id, TagEOF, []byte{})
}
