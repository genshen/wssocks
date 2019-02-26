package ws_socks

import (
	"errors"
	"github.com/gorilla/websocket"
)

// version of protocol.
const VersionCode = 0x001

type VersionNeg struct {
	Version     string `json:"version"`
	VersionCode uint   `json:"version_code"`
	UpdateAddr  string `json:"update_addr"`
}

// negotiate client and server version
// after websocket connection is established,
// client can receive a message from server with server version number.
func NegVersionClient(wsConn *websocket.Conn) (VersionNeg, error) {
	var versionData VersionNeg
	if err := wsConn.ReadJSON(&versionData); err != nil {
		return versionData, err
	}
	if versionData.VersionCode != VersionCode {
		return versionData, errors.New("incompatible protocol version of client and server")
	}
	return versionData, nil
}

// send version information to client from server
func NegVersionServer(wsConn *websocket.Conn) error {
	versionData := VersionNeg{VersionCode: VersionCode}
	return wsConn.WriteJSON(&versionData)
}
