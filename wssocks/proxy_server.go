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
	Conn      *net.TCPConn
	Id        ksuid.KSUID
	closeable bool
}

// proxy server, which handles many tcp connection
type ServerWS struct {
	ConcurrentWebSocket
	mu    sync.RWMutex
	conns map[ksuid.KSUID]*Connector
}

func (s *ServerWS) NewConn(id ksuid.KSUID, conn *net.TCPConn) *Connector {
	s.mu.Lock()
	defer s.mu.Unlock()
	connector := Connector{Id: id, Conn: conn, closeable: true}
	s.conns[id] = &connector
	return &connector
}

func (s *ServerWS) GetConnectorById(id ksuid.KSUID) *Connector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if connector, ok := s.conns[id]; ok {
		return connector
	}
	return nil
}

func (s *ServerWS) GetConnectorSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.conns)
}

func (s *ServerWS) Close(id ksuid.KSUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if connector, ok := s.conns[id]; ok {
		if connector.closeable { // todo set closeable to false in map
			if err := connector.Conn.Close(); err != nil {
				delete(s.conns, id)
				return err
			}
		}
		delete(s.conns, id)
	}
	return nil
}

// in this case, one ws only handle one proxy.
func (s *ServerWS) dispatchMessage(data []byte) error {
	var socketData json.RawMessage
	socketStream := WebSocketMessage{
		Data: &socketData,
	}
	if err := json.Unmarshal(data, &socketStream); err != nil {
		return nil // skip error
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
		s.Close(id)
	case WsTpEst: // establish
		var proxyMsg ProxyMessage
		if err := json.Unmarshal(socketData, &proxyMsg); err != nil {
			return nil
		} else {
			go func() {
				s.establish(id, proxyMsg.Addr) // todo error handle
				s.tellClosed(id)
			}()
		}
	case WsTpData:
		var requestMsg RequestMessage
		if err := json.Unmarshal(socketData, &requestMsg); err != nil {
			return nil
		}

		if connector := s.GetConnectorById(id); connector != nil {
			go func() {
				// write income data from websocket to TCP connection
				if decodeBytes, err := base64.StdEncoding.DecodeString(requestMsg.DataBase64); err != nil {
					log.Println("base64 decode error,", err)
					// return err
				} else {
					if _, err := connector.Conn.Write(decodeBytes); err != nil {
						s.tellClosed(id)
						s.Close(id) // also closed= tcp connection if it exists
						//return err
					}
				}
			}()
		}
		return nil
	}
	return nil
}

// the the client the connection has been closed
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

func (s *ServerWS) establish(id ksuid.KSUID, addr string) error {
	log.Println("info", "proxy to:", addr)
	tcpConn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		log.Println(err)
		return err
	}

	connector := s.NewConn(id, tcpConn.(*net.TCPConn))
	defer log.Println("info", "disconnected to:", addr)
	defer s.Close(id)

	if proxyServerTicker != nil {
		var sendBuffer Base64WSBufferWriter
		if _, err := sendBuffer.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
		log.Println("info", "connected to:", addr)

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
		if err := s.WriteMessage(id, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
		var buffer = make([]byte, 1024*64)
		for {
			if n, err := connector.Conn.Read(buffer); err != nil {
				log.Println("read error:", err)
				break
			} else if n > 0 {
				if err := s.WriteMessage(id, buffer[:n]); err != nil {
					log.Println("write websocket error:", err)
					break
				}
			}
		}
	}
	return nil
}
