package ws_socks

import (
	"bytes"
	"encoding/base64"
	"github.com/segmentio/ksuid"
	"sync"
)

// copy data from WebSocket to ssh server
// and copy data from ssh server to WebSocket

// write data to WebSocket
// the data comes from ssh server.
type Base64WSBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (b *Base64WSBufferWriter) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

// flush all data in this buff into WebSocket.
func (b *Base64WSBufferWriter) Flush(messageType int, id ksuid.KSUID, cwi ConcurrentWebSocketInterface) (int, error) {
	length := b.buffer.Len()
	if length != 0 {
		b.mu.Lock()
		defer b.mu.Unlock()

		dataBase64 := base64.StdEncoding.EncodeToString(b.buffer.Bytes())
		jsonData := WebSocketMessage2{
			Id:   id.String(),
			Type: WsTpData,
			Data: RequestMessage{DataBase64: dataBase64},
		}
		err := cwi.WriteWSJSON(&jsonData)
		if err != nil {
			return 0, err
		}
		b.buffer.Reset()
	}
	return length, nil
}
