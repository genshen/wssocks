package wss

import (
	"context"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// version of protocol.
const VersionCode = 0x004
const CompVersion = 0x003
const CoreVersion = "0.6.0"

type VersionNeg struct {
	Version          string `json:"version"`
	CompVersion      uint   `json:"comp_version"` // Compatible version code
	VersionCode      uint   `json:"version_code"`
	EnableStatusPage bool   `json:"status_page"`
}

// negotiate client and server version
// after websocket connection is established,
// client can receive a message from server with server version number.
func ExchangeVersion(ctx context.Context, wsConn *websocket.Conn) (VersionNeg, error) {
	var versionRec VersionNeg
	versionServer := VersionNeg{Version: CoreVersion, VersionCode: VersionCode}
	if err := wsjson.Write(ctx, wsConn, &versionServer); err != nil {
		return versionRec, err
	}
	if err := wsjson.Read(ctx, wsConn, &versionRec); err != nil {
		return versionRec, err
	}
	return versionRec, nil
}

// send version information to client from server
func NegVersionServer(ctx context.Context, wsConn *websocket.Conn, enableStatusPage bool) error {
	// read from client
	var versionClient VersionNeg
	if err := wsjson.Read(ctx, wsConn, &versionClient); err != nil {
		return err
	}
	// send to client
	versionServer := VersionNeg{
		Version:          CoreVersion,
		CompVersion:      CompVersion,
		VersionCode:      VersionCode,
		EnableStatusPage: enableStatusPage,
	} // todo more information
	return wsjson.Write(ctx, wsConn, &versionServer)
}
