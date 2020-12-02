package wss

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"nhooyr.io/websocket"
	"time"
)

type Connector struct {
	Conn io.ReadWriteCloser
}

// interface of establishing proxy connection with target
type ProxyEstablish interface {
	establish(hub *Hub, id ksuid.KSUID, proxyType int, addr string, data []byte) error

	// data from client todo data with type
	onData(data ClientData) error

	// close connection
	// tell: whether to send close message to proxy client
	Close(tell bool) error
}

type ClientData ServerData

var ConnCloseByClient = errors.New("conn closed by client")

func dispatchMessage(hub *Hub, msgType websocket.MessageType, data []byte, config WebsocksServerConfig) error {
	if msgType == websocket.MessageText {
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
		return hub.CloseProxyConn(id)
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
		go establishProxy(hub, ProxyRegister{id, proxyEstMsg.Type, proxyEstMsg.Addr, estData})
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
				return proxy.ProxyIns.onData(ClientData{Tag: requestMsg.Tag, Data: decodeBytes})
			}
		}
		return nil
	}
	return nil
}

func establishProxy(hub *Hub, proxyMeta ProxyRegister) {
	var e ProxyEstablish
	if proxyMeta._type == ProxyTypeHttp {
		e = &HttpProxyEst{}
	} else {
		e = &DefaultProxyEst{}
	}

	err := e.establish(hub, proxyMeta.id, proxyMeta._type, proxyMeta.addr, proxyMeta.withData)
	if err == nil {
		hub.tellClosed(proxyMeta.id) // tell client to close connection.
	} else if err != ConnCloseByClient {
		log.Error(err) // todo error handle better way
		hub.tellClosed(proxyMeta.id)
	}
	return
	//	log.WithField("size", s.GetConnectorSize()).Trace("connection size changed.")
}

// data type used in DefaultProxyEst to pass data to channel
type ChanDone struct {
	tell bool
	err  error
}

// interface implementation for socks5 and https proxy.
type DefaultProxyEst struct {
	done    chan ChanDone
	tcpConn net.Conn
}

func (e *DefaultProxyEst) onData(data ClientData) error {
	if _, err := e.tcpConn.Write(data.Data); err != nil {
		e.done <- ChanDone{true, err}
	}
	return nil
}

func (e *DefaultProxyEst) Close(tell bool) error {
	e.done <- ChanDone{tell, ConnCloseByClient}
	return nil // todo error
}

// data: data send in establish step (can be nil).
func (e *DefaultProxyEst) establish(hub *Hub, id ksuid.KSUID, proxyType int, addr string, data []byte) error {
	conn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}
	e.tcpConn = conn
	defer conn.Close()

	e.done = make(chan ChanDone, 2)
	//defer close(done)

	// todo check exists
	hub.addNewProxy(&ProxyServer{Id: id, ProxyIns: e})
	defer hub.RemoveProxy(id)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	switch proxyType {
	case ProxyTypeSocks5:
		if err := hub.WriteProxyMessage(ctx, id, TagData, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
	case ProxyTypeHttps:
		if err := hub.WriteProxyMessage(ctx, id, TagData, []byte("HTTP/1.0 200 Connection Established\r\nProxy-agent: wssocks\r\n\r\n")); err != nil {
			return err
		}
	}

	go func() {
		writer := NewWebSocketWriter(&hub.ConcurrentWebSocket, id, context.Background())
		if _, err := io.Copy(writer, conn); err != nil {
			log.Error("copy error,", err)
			e.done <- ChanDone{true, err}
		}
		e.done <- ChanDone{true, nil}
	}()

	d := <-e.done
	// s.RemoveProxy(proxy.Id)
	// tellClosed is called outside this func.
	return d.err
}

type HttpProxyEst struct {
	bodyReadCloser *BufferedWR
}

func (h *HttpProxyEst) onData(data ClientData) error {
	if data.Tag == TagNoMore {
		return h.bodyReadCloser.Close() // close due to no more data.
	}
	if _, err := h.bodyReadCloser.Write(data.Data); err != nil {
		return err
	}
	return nil
}

func (h *HttpProxyEst) Close(tell bool) error {
	return h.bodyReadCloser.Close() // close from client
}

func (h *HttpProxyEst) establish(hub *Hub, id ksuid.KSUID, proxyType int, addr string, header []byte) error {
	if header == nil {
		hub.tellClosed(id)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = hub.WriteProxyMessage(ctx, id, TagEstErr, nil)
		return errors.New("http header empty")
	}

	closed := make(chan bool)
	client := make(chan ClientData, 2) // for http at most 2 data buffers are needed(http body, TagNoMore tag).
	defer close(closed)
	defer close(client)

	hub.addNewProxy(&ProxyServer{Id: id, ProxyIns: h})
	bodyReadCloser := NewBufferWR()
	defer hub.RemoveProxy(id)
	defer func() {
		if !bodyReadCloser.isClosed() { // if it is not closed by client.
			hub.tellClosed(id) // todo
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := hub.ConcurrentWebSocket.WriteProxyMessage(ctx, id, TagEstOk, nil); err != nil {
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

	writer := NewWebSocketWriter(&hub.ConcurrentWebSocket, id, context.Background())
	var headerBuffer bytes.Buffer
	HttpRespHeader(&headerBuffer, resp)
	writer.Write(headerBuffer.Bytes())
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("http body copy error: %w", err)
	}
	return nil
}
