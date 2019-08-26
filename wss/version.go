package wss

import (
	"github.com/gorilla/websocket"
)

// version of protocol.
const VersionCode = 0x002
const CoreVersion = "0.3.0-alpha"

type VersionNeg struct {
	Version     string `json:"version"`
	VersionCode uint   `json:"version_code"`
}

// negotiate client and server version
// after websocket connection is established,
// client can receive a message from server with server version number.
func ExchangeVersion(wsConn *websocket.Conn) (VersionNeg, error) {
	var versionRec VersionNeg
	versionServer := VersionNeg{VersionCode: VersionCode}
	if err := wsConn.WriteJSON(&versionServer); err != nil {
		return versionRec, err
	}
	if err := wsConn.ReadJSON(&versionRec); err != nil {
		return versionRec, err
	}
	return versionRec, nil
}

// send version information to client from server
func NegVersionServer(wsConn *websocket.Conn) error {
	// read from client
	var versionClient VersionNeg
	if err := wsConn.ReadJSON(&versionClient); err != nil {
		return err
	}
	// send to client
	versionServer := VersionNeg{Version: CoreVersion, VersionCode: VersionCode} // todo more information
	return wsConn.WriteJSON(&versionServer)
}
