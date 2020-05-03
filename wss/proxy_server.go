package wss

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"time"
)

type Connector struct {
	Conn io.ReadWriteCloser
}

type ClientData ServerData

func dispatchMessage(hub *Hub, msgType int, data []byte, config WebsocksServerConfig) error {
	if msgType == websocket.TextMessage {
		return dispatchDataMessage(hub, data, config)
	}
	return nil
}

func dispatchDataMessage(hub *Hub, data []byte, config WebsocksServerConfig) error {
	var socketData json.RawMessage
	socketStream := WebSocketMessage{
		Data: &socketData,
	}
	if err := json.Unmarshal(data, &socketStream); err != nil {
		return err
	}

	// parsing id
	id, err := ksuid.Parse(socketStream.Id)
	if err != nil {
		return err
	}

	switch socketStream.Type {
	case WsTpBeats: // heart beats
	case WsTpClose: // closed by client
		hub.unregister <- id
		return nil
	case WsTpEst: // establish
		var proxyEstMsg ProxyEstMessage
		if err := json.Unmarshal(socketData, &proxyEstMsg); err != nil {
			return err
		}
		// check proxy type support.
		if (proxyEstMsg.Type == ProxyTypeHttp || proxyEstMsg.Type == ProxyTypeHttps) && !config.EnableHttp {
			hub.tellClosed(id) // tell client to close connection.
			return errors.New("http(s) proxy is not support in server side")
		}

		var estData []byte = nil
		if proxyEstMsg.WithData {
			if decodedBytes, err := base64.StdEncoding.DecodeString(proxyEstMsg.DataBase64); err != nil {
				log.Error("base64 decode error,", err)
				return err
			} else {
				estData = decodedBytes
			}
		}
		hub.est <- ProxyRegister{id, proxyEstMsg.Type, proxyEstMsg.Addr, estData}
	case WsTpData:
		var requestMsg ProxyData
		if err := json.Unmarshal(socketData, &requestMsg); err != nil {
			return err
		}

		if proxy := hub.GetProxyById(id); proxy != nil {
			// write income data from websocket to TCP connection
			if decodeBytes, err := base64.StdEncoding.DecodeString(requestMsg.DataBase64); err != nil {
				log.Error("base64 decode error,", err)
				return err
			} else {
				proxy.onData(ClientData{Tag: requestMsg.Tag, Data: decodeBytes})
			}
		}
		return nil
	}
	return nil
}

func establishProxy(hub *Hub, proxyMeta ProxyRegister) {
	// todo size is only one client's size.
	//	log.WithField("size", s.GetConnectorSize()+1).Trace("connection size changed.")
	//	log.WithField("address", proxyEstMsg.Addr).Trace("proxy connected to remote")
	if proxyMeta._type == ProxyTypeHttp {
		if err := establishHttp(hub, proxyMeta.id, proxyMeta._type, proxyMeta.addr, proxyMeta.withData); err != nil {
			log.Error(err) // todo error handle better way
		}
	} else {
		if err := establish(hub, proxyMeta.id, proxyMeta._type, proxyMeta.addr, proxyMeta.withData); err != nil {
			log.Error(err) // todo error handle better way
		}
	}
	//	log.WithField("size", s.GetConnectorSize()).Trace("connection size changed.")
	hub.tellClose <- proxyMeta.id // tell client to close connection.
}

// data: data send in establish step (can be nil).
func establish(hub *Hub, id ksuid.KSUID, proxyType int, addr string, data []byte) error {
	tcpConn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	type Done struct {
		tell bool
		err  error
	}
	done := make(chan Done, 2)
	//defer close(done)

	// todo check exists
	proxy := ProxyServer{Id: id, onData: func(data ClientData) {
		if _, err := tcpConn.Write(data.Data); err != nil {
			done <- Done{true, err}
		}
	}, onClosed: func(tell bool) {
		done <- Done{tell, err}
	}}
	hub.register <- &proxy
	defer hub.RemoveProxy(id)

	switch proxyType {
	case ProxyTypeSocks5:
		if err := hub.WriteProxyMessage(id, TagData, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
	case ProxyTypeHttps:
		if err := hub.WriteProxyMessage(id, TagData, []byte("HTTP/1.0 200 Connection Established\r\nProxy-agent: wssocks\r\n\r\n")); err != nil {
			return err
		}
	}

	go func() {
		writer := WebSocketWriter{WSC: &hub.ConcurrentWebSocket, Id: id}
		if _, err := io.Copy(&writer, tcpConn); err != nil {
			log.Error("copy error,", err)
		}
		done <- Done{true, nil}
	}()

	d := <-done
	// s.RemoveProxy(proxy.Id)
	// tellClosed is called outside this func.
	if d.err != nil {
		return d.err
	}
	return nil
}

func establishHttp(hub *Hub, id ksuid.KSUID, proxyType int, addr string, header []byte) error {
	if header == nil {
		hub.tellClose <- id
		_ = hub.WriteProxyMessage(id, TagEstErr, nil)
		return errors.New("http header empty")
	}

	closed := make(chan bool)
	client := make(chan ClientData, 2) // for http at most 2 data buffers are needed(http body, TagNoMore tag).
	defer close(closed)
	defer close(client)

	bodyReadCloser := NewBufferWR()
	proxy := ProxyServer{Id: id, onData: func(data ClientData) {
		if data.Tag == TagNoMore {
			bodyReadCloser.Close() // close due to no more data.
			return
		}
		bodyReadCloser.Write(data.Data)
	}, onClosed: func(tell bool) {
		bodyReadCloser.Close() // close from client
	}}
	hub.register <- &proxy
	defer hub.RemoveProxy(id)
	defer func() {
		if !bodyReadCloser.isClosed() { // if it is not closed by client.
			hub.tellClose <- id
		}
	}()

	if err := hub.WriteProxyMessage(id, TagEstOk, nil); err != nil {
		return err
	}

	// get http request by header bytes.
	bufferHeader := bufio.NewReader(bytes.NewBuffer(header))
	req, err := http.ReadRequest(bufferHeader)
	if err != nil {
		return err
	}
	req.Body = bodyReadCloser

	// read request and copy response back
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("transport error: %w", err)
	}
	defer resp.Body.Close()

	writer := WebSocketWriter{WSC: &hub.ConcurrentWebSocket, Id: id}
	var headerBuffer bytes.Buffer
	HttpRespHeader(&headerBuffer, resp)
	writer.Write(headerBuffer.Bytes())
	if _, err := io.Copy(&writer, resp.Body); err != nil {
		return fmt.Errorf("http body copy error: %w", err)
	}
	return nil
}
