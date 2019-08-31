package wss

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
	"time"
)

type Connector struct {
	Conn io.ReadWriteCloser
}

type ClientData ServerData

type ProxyServer struct {
	Id     ksuid.KSUID
	client chan ClientData // data from client todo data with type
	close  chan bool       // close connection by this channel
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
func (s *ServerWS) AddConn(id ksuid.KSUID, client chan ClientData, close chan bool) *ProxyServer {
	s.mu.Lock()
	defer s.mu.Unlock()
	proxy := ProxyServer{Id: id, client: client, close: close}
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
	id, err := ksuid.Parse(socketStream.Id);
	if err != nil {
		return err
	}

	switch socketStream.Type {
	case WsTpBeats: // heart beats
	case WsTpClose: // closed by client
		if proxy := s.GetProxyById(id); proxy != nil {
			proxy.close <- false
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
			log.WithField("address", proxyEstMsg.Addr).Trace("proxy connecting to remote")
			if err := s.establish(id, proxyEstMsg.Type, proxyEstMsg.Addr, estData); err != nil {
				log.Error(err) // todo error handle better way
			}
			log.WithField("address", proxyEstMsg.Addr).Trace("disconnected to remote")
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
				proxy.client <- ClientData{Type: requestMsg.Type, Data: decodeBytes}
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

	closed := make(chan bool)
	client := make(chan ClientData)
	defer close(closed)
	defer close(client)

	// todo check exists
	proxy := s.AddConn(id, client, closed)
	defer s.RemoveProxy(id)

	switch proxyType {
	case ProxyTypeSocks5:
		if err := s.WriteProxyMessage(id, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
	case ProxyTypeHttp:
		if data != nil {
			if _, err := tcpConn.Write(data); err != nil {
				return err
			}
		}
	case ProxyTypeHttps:
		if err := s.WriteProxyMessage(id, []byte("HTTP/1.0 200 Connection Established\r\nProxy-agent: Pyx\r\n\r\n")); err != nil {
			return err
		}
	}

	go func() {
		writer := WebSocketWriter{WSC: &s.ConcurrentWebSocket, Id: proxy.Id}
		if _, err := io.Copy(&writer, tcpConn); err != nil {
			log.Error("copy error,", err)
		}
		closed <- true
	}()

	for {
		select {
		case tellClose := <-closed:
			if tellClose {
				return s.tellClosed(proxy.Id)
			}
			return nil
		case client := <-client: // data received from client
			if _, err := tcpConn.Write(client.Data); err != nil {
				s.tellClosed(proxy.Id)
				return err
			}
		}
	}
}
