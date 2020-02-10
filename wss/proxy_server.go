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
	"sync"
	"time"
)

type Connector struct {
	Conn io.ReadWriteCloser
}

type ClientData ServerData

type ProxyServer struct {
	Id       ksuid.KSUID
	onData   func(ClientData) // data from client todo data with type
	onClosed func(bool)       // close connection, param bool: do tellClose if true
}

// proxy server, which handles many tcp connection
type ServerWS struct {
	ConcurrentWebSocket
	mu       sync.RWMutex
	connPool map[ksuid.KSUID]*ProxyServer
}

// create a new websocket server handler
func NewServerWS(conn *websocket.Conn) *ServerWS {
	sws := ServerWS{ConcurrentWebSocket: ConcurrentWebSocket{WsConn: conn}}
	sws.connPool = make(map[ksuid.KSUID]*ProxyServer)
	return &sws
}

// add a tcp connection to connection pool.
func (s *ServerWS) AddConn(id ksuid.KSUID, onData func(ClientData), onClosed func(bool)) *ProxyServer {
	s.mu.Lock()
	defer s.mu.Unlock()
	proxy := ProxyServer{Id: id, onData: onData, onClosed: onClosed}
	s.connPool[id] = &proxy
	return &proxy
}

func (s *ServerWS) GetProxyById(id ksuid.KSUID) *ProxyServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if proxy, ok := s.connPool[id]; ok {
		return proxy
	}
	return nil
}

func (s *ServerWS) GetConnectorSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connPool)
}

// remove a connection specified by id.
func (s *ServerWS) RemoveProxy(id ksuid.KSUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.connPool[id]; ok {
		delete(s.connPool, id)
	}
}

// remove all connections in pool
func (s *ServerWS) RemoveAll(id ksuid.KSUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.connPool {
		delete(s.connPool, id)
	}
}

// tell the client the connection has been closed
func (s *ServerWS) tellClosed(id ksuid.KSUID) error {
	// send finish flag to client
	finish := WebSocketMessage{
		Id:   id.String(),
		Type: WsTpClose,
		Data: nil,
	}
	if err := s.WriteWSJSON(&finish); err != nil {
		return err
	}
	return nil
}

func (s *ServerWS) dispatchMessage(data []byte, config WebsocksServerConfig) error {
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
		if proxy := s.GetProxyById(id); proxy != nil {
			proxy.onClosed(false)
		}
		return nil
	case WsTpEst: // establish
		var proxyEstMsg ProxyEstMessage
		if err := json.Unmarshal(socketData, &proxyEstMsg); err != nil {
			return err
		}
		// check proxy type support.
		if (proxyEstMsg.Type == ProxyTypeHttp || proxyEstMsg.Type == ProxyTypeHttps) && !config.EnableHttp {
			s.tellClosed(id) // tell client to close connection.
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

		go func() {
			// todo size is only one client's size.
			log.WithField("size", s.GetConnectorSize()+1).Trace("connection size changed.")
			log.WithField("address", proxyEstMsg.Addr).Trace("proxy connected to remote")
			if proxyEstMsg.Type == ProxyTypeHttp {
				if err := s.establishHttp(id, proxyEstMsg.Type, proxyEstMsg.Addr, estData); err != nil {
					log.Error(err) // todo error handle better way
				}
			} else {
				if err := s.establish(id, proxyEstMsg.Type, proxyEstMsg.Addr, estData); err != nil {
					log.Error(err) // todo error handle better way
				}
			}
			log.WithField("size", s.GetConnectorSize()).Trace("connection size changed.")
			s.tellClosed(id) // tell client to close connection.
		}()
	case WsTpData:
		var requestMsg ProxyData
		if err := json.Unmarshal(socketData, &requestMsg); err != nil {
			return err
		}

		if proxy := s.GetProxyById(id); proxy != nil {
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

// data: data send in establish step (can be nil).
func (s *ServerWS) establish(id ksuid.KSUID, proxyType int, addr string, data []byte) error {
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
	proxy := s.AddConn(id, func(data ClientData) {
		if _, err := tcpConn.Write(data.Data); err != nil {
			done <- Done{true, err}
		}
	}, func(tell bool) {
		done <- Done{tell, err}
	})
	defer s.RemoveProxy(id)

	switch proxyType {
	case ProxyTypeSocks5:
		if err := s.WriteProxyMessage(id, TagData, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
	case ProxyTypeHttps:
		if err := s.WriteProxyMessage(id, TagData, []byte("HTTP/1.0 200 Connection Established\r\nProxy-agent: Pyx\r\n\r\n")); err != nil {
			return err
		}
	}

	go func() {
		writer := WebSocketWriter{WSC: &s.ConcurrentWebSocket, Id: proxy.Id}
		if _, err := io.Copy(&writer, tcpConn); err != nil {
			log.Error("copy error,", err)
		}
		done <- Done{true, nil}
	}()

	d := <-done
	s.RemoveProxy(proxy.Id)
	// tellClosed is called outside this func.
	if d.err != nil {
		return d.err
	}
	return nil
}

func (s *ServerWS) establishHttp(id ksuid.KSUID, proxyType int, addr string, header []byte) error {
	if header == nil {
		_ = s.tellClosed(id)
		_ = s.WriteProxyMessage(id, TagEstErr, nil)
		return errors.New("http header empty")
	}

	closed := make(chan bool)
	client := make(chan ClientData, 2) // for http at most 2 data buffers are needed(http body, TagNoMore tag).
	defer close(closed)
	defer close(client)

	bodyReadCloser := NewBufferWR()
	proxy := s.AddConn(id, func(data ClientData) {
		if data.Tag == TagNoMore {
			bodyReadCloser.Close() // close due to no more data.
			return
		}
		bodyReadCloser.Write(data.Data)
	}, func(tell bool) {
		bodyReadCloser.Close() // close from client
	})
	defer s.RemoveProxy(id)
	defer func() {
		if !bodyReadCloser.isClosed() { // if it is not closed by client.
			_ = s.tellClosed(id)
		}
	}()

	if err := s.WriteProxyMessage(id, TagEstOk, nil); err != nil {
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

	writer := WebSocketWriter{WSC: &s.ConcurrentWebSocket, Id: proxy.Id}
	var headerBuffer bytes.Buffer
	HttpRespHeader(&headerBuffer, resp)
	writer.Write(headerBuffer.Bytes())
	if _, err := io.Copy(&writer, resp.Body); err != nil {
		return fmt.Errorf("http body copy error: %w", err)
	}
	return nil
}
