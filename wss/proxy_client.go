package wss

import (
	"encoding/base64"
	"github.com/genshen/wssocks/wss/term_view"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

// proxy client handle one connection, send data to proxy server vai websocket.
type ProxyClient struct {
	Id       ksuid.KSUID
	onData   func(ksuid.KSUID, ServerData) // data from server todo data with  type
	onClosed func(ksuid.KSUID, bool)       // close connection, param bool: do tellClose if true
	onError  func(ksuid.KSUID, error)      // if there are error messages
}

type ServerData struct {
	Tag  int
	Data []byte
}

// handel socket dial results processing
// copy income connection data to proxy serve via websocket
func (p *ProxyClient) Establish(plog *term_view.ProgressLog, wsc *WebSocketClient,
	firstSendData []byte, proxyType int, addr string) error {
	estMsg := ProxyEstMessage{
		Type:     proxyType,
		Addr:     addr,
		WithData: false,
	}
	if firstSendData != nil {
		estMsg.WithData = true
		estMsg.DataBase64 = base64.StdEncoding.EncodeToString(firstSendData)
	}
	addrSend := WebSocketMessage{Type: WsTpEst, Id: p.Id.String(), Data: estMsg}
	if err := wsc.WriteWSJSON(&addrSend); err != nil {
		log.Error("json error:", err)
		return err
	}
	return nil
}
