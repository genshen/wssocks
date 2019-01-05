package ws_socks

import (
	"bytes"
	"encoding/base64"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

// copy data from WebSocket to ssh server
// and copy data from ssh server to WebSocket

// write data to WebSocket
// the data comes from ssh server.
type WebSocketBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (w *WebSocketBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// flush all data in this buff into WebSocket.
func (w *WebSocketBufferWriter) Flush(messageType int, ws *websocket.Conn) error {
	if w.buffer.Len() != 0 {
		w.mu.Lock()
		defer w.mu.Unlock()
		err := ws.WriteMessage(messageType, w.buffer.Bytes())
		if err != nil {
			return err
		}
		log.Println("write buffer", w.buffer.Len())
		w.buffer.Reset()
	}
	return nil
}

type Base64WSBufferWriter struct {
	WebSocketBufferWriter
}

func (b *Base64WSBufferWriter) Flush(messageType int, ws *websocket.Conn) error {
	if b.buffer.Len() != 0 {
		b.mu.Lock()
		defer b.mu.Unlock()

		dataBase64 := base64.StdEncoding.EncodeToString(b.buffer.Bytes())
		jsonData := WebSocketMessage{
			Type: WebSocketMessageTypeRequest,
			Data: RequestMessage{DataBase64: dataBase64},
		}
		err := ws.WriteJSON(&jsonData)
		if err != nil {
			return err
		}
		log.Println("flush length ", b.buffer.Len())
		b.buffer.Reset()
	}
	return nil
}
