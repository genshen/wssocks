package ws_socks

import (
	"encoding/base64"
	"encoding/json"
	"github.com/genshen/ws-socks/wssocks/ticker"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// ticker for proxy server
var proxyServerTicker *ticker.Ticker = nil

func StartTicker(d time.Duration) *ticker.Ticker {
	proxyServerTicker = ticker.NewTicker()
	proxyServerTicker.Start(d)
	return proxyServerTicker
}

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

func (s *ServerWS) dispatchMessage(data []byte) error {
	var socketData json.RawMessage
	socketStream := WebSocketMessage{
		Data: &socketData,
	}
	if err := json.Unmarshal(data, &socketStream); err != nil {
		return err
	}

	// parsing id
	var id ksuid.KSUID
	if _id, err := ksuid.Parse(socketStream.Id); err != nil {
		return err
	} else {
		id = _id
	}

	switch socketStream.Type {
	case WsTpBeats: // heart beats
	case WsTpClose: // closed by client
		return s.Close(id)
	case WsTpEst: // establish
		var proxyEstMsg ProxyEstMessage
		if err := json.Unmarshal(socketData, &proxyEstMsg); err != nil {
			return err
		} else {
			go func() {
				log.Println("info", "proxy to:", proxyEstMsg.Addr)
				if err := s.establish(id, proxyEstMsg.Addr); err != nil {
					log.Println(err) // todo error handle better way
				}
				log.Println("info", "disconnected to:", proxyEstMsg.Addr)
				s.tellClosed(id) // tell client to close connection.
			}()
		}
	case WsTpData:
		var requestMsg ProxyData
		if err := json.Unmarshal(socketData, &requestMsg); err != nil {
			return err
		}

		if connector := s.GetConnectorById(id); connector != nil {
			//go func() {
			// write income data from websocket to TCP connection
			if decodeBytes, err := base64.StdEncoding.DecodeString(requestMsg.DataBase64); err != nil {
				log.Println("base64 decode error,", err)
				return err
			} else {
				if _, err := connector.Conn.Write(decodeBytes); err != nil {
					s.tellClosed(id)
					return s.Close(id) // also closed= tcp connection if it exists
					// todo return err
				}
			}
			// }()
		}
		return nil
	}
	return nil
}

func (s *ServerWS) establish(id ksuid.KSUID, addr string) error {
	tcpConn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}

	// todo check exists
	connector := s.AddConn(id, tcpConn.(*net.TCPConn))
	defer s.Close(id)

	if proxyServerTicker != nil {
		var sendBuffer Base64WSBufferWriter
		if _, err := sendBuffer.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}

		defer sendBuffer.Flush(websocket.TextMessage, id, &(s.ConcurrentWebSocket))
		defer proxyServerTicker.Remove(ticker.TickId(id))

		proxyServerTicker.Append(ticker.TickId(id), func() {
			// fixme return error
			if _, err := sendBuffer.Flush(websocket.TextMessage, id, &(s.ConcurrentWebSocket)); err != nil {
				log.Println("Error: error sending data via webSocket:", err)
				return
			}
		})

		if _, err := io.Copy(&sendBuffer, connector.Conn); err != nil {
			return err
		}
	} else {
		// no ticker
		if err := s.WriteProxyMessage(id, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
		var buffer = make([]byte, 1024*64)
		for {
			if n, err := connector.Conn.Read(buffer); err != nil {
				log.Println("read error:", err)
				break
			} else if n > 0 {
				if err := s.WriteProxyMessage(id, buffer[:n]); err != nil {
					log.Println("write websocket error:", err)
					break
				}
			}
		}
	}
	return nil
}
