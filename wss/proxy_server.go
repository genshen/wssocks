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
	Conn *net.TCPConn
}

// proxy server, which handles many tcp connection
type ServerWS struct {
	ConcurrentWebSocket
	mu       sync.RWMutex
	connPool map[ksuid.KSUID]*Connector
}

// create a new websocket server handler
func NewServerWS(conn *websocket.Conn) *ServerWS {
	sws := ServerWS{ConcurrentWebSocket: ConcurrentWebSocket{WsConn: conn}}
	sws.connPool = make(map[ksuid.KSUID]*Connector)
	return &sws
}

// add a tcp connection to connection pool.
func (s *ServerWS) AddConn(id ksuid.KSUID, conn *net.TCPConn) *Connector {
	s.mu.Lock()
	defer s.mu.Unlock()
	connector := Connector{Conn: conn}
	s.connPool[id] = &connector
	return &connector
}

func (s *ServerWS) GetConnectorById(id ksuid.KSUID) *Connector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if connector, ok := s.connPool[id]; ok {
		return connector
	}
	return nil
}

func (s *ServerWS) GetConnectorSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connPool)
}

// close a connection specified by id.
func (s *ServerWS) Close(id ksuid.KSUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if connector, ok := s.connPool[id]; ok {
		err := connector.Conn.Close();
		delete(s.connPool, id)
		return err
	}
	return nil
}

// close all connections in pool
func (s *ServerWS) CloseAll(id ksuid.KSUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	for id, conn := range s.connPool {
		if err != nil {
			_ = conn.Conn.Close()
		} else {
			err = conn.Conn.Close() // set error as return
		}
		delete(s.connPool, id)
	}
	return err
}

// tell the client the connection has been closed
func (s *ServerWS) tellClosed(id ksuid.KSUID) {
	// send finish flag to client
	finish := WebSocketMessage{
		Id:   id.String(),
		Type: WsTpClose,
		Data: nil,
	}
	if err := s.WriteWSJSON(&finish); err != nil {
		return
	}
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
		return s.Close(id)
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

		if connector := s.GetConnectorById(id); connector != nil {
			//go func() {
			// write income data from websocket to TCP connection
			if decodeBytes, err := base64.StdEncoding.DecodeString(requestMsg.DataBase64); err != nil {
				log.Error("base64 decode error,", err)
				return err
			} else {
				if _, err := connector.Conn.Write(decodeBytes); err != nil {
					s.tellClosed(id)
					return s.Close(id) // also closed= tcp connection if it exists
				}
			}
			// }()
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

	// todo check exists
	connector := s.AddConn(id, tcpConn.(*net.TCPConn))
	defer s.Close(id)

	switch proxyType {
	case ProxyTypeSocks5:
		if err := s.WriteProxyMessage(id, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
	case ProxyTypeHttp:
		if data != nil {
			if _, err := connector.Conn.Write(data); err != nil {
				return err
			}
		}
	case ProxyTypeHttps:
		if err := s.WriteProxyMessage(id, []byte("HTTP/1.0 200 Connection Established\r\nProxy-agent: Pyx\r\n\r\n")); err != nil {
			return err
		}
	}

	var buffer = make([]byte, 1024*64)
	for {
		if n, err := connector.Conn.Read(buffer); err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.New("read connection error:" + err.Error())
		} else if n > 0 {
			if err := s.WriteProxyMessage(id, buffer[:n]); err != nil {
				return errors.New("error sending data via webSocket:" + err.Error())
			}
		}
	}
	return nil
}
