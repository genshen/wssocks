package ws_socks

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestWsDataType(t *testing.T) {
	// encode
	s := "abcd"
	var v = []byte(s)

	dataBase64 := base64.StdEncoding.EncodeToString(v)
	jsonData := WebSocketMessage{
		Type: WebSocketMessageTypeRequest,
		Data: RequestMessage{DataBase64: string(dataBase64)},
	}
	fmt.Println(jsonData)
}
