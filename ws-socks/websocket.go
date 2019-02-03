package ws_socks

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const TimerDuration = 60                  // millisecond
const TimeOuts = 6 * 1000 / TimerDuration // 6 seconds

var upgrader = websocket.Upgrader{} // use default options

// listen http port and serve it
// serveWs handles websocket requests from the peer.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		log.Println("Error: Not a websocket handshake", 400)
		return
	} else if err != nil {
		http.Error(w, "Cannot setup WebSocket connection:", 400)
		log.Println("Error: Cannot setup WebSocket connection:", err)
		return
	}
	defer ws.Close()

	done := make(chan bool, 1)
	setDone := func() { done <- true }

	sws := ServerWS{ConcurrentWebSocket: ConcurrentWebSocket{WsConn: ws}}
	sws.conns = make(map[ksuid.KSUID]*Connector)
	go func() { // read messages from webSocket
		defer setDone()
		for {
			log.Println("conn size: ", sws.GetConnectorSize())
			_, p, err := ws.ReadMessage()
			// if WebSocket is closed by some reason, then this func will return,
			// and 'done' channel will be set, the outer func will reach to the end.
			// then ssh session will be closed in defer.
			if err != nil {
				log.Println("Error: error reading webSocket message:", err)
				return
			}
			if err = sws.dispatchMessage(p); err != nil { // todo go
				log.Println("Error: error proxy:", err)
				return
			}
		}
	}()

	<-done
}

type Connector struct {
	sendBuffer Base64WSBufferWriter
	Conn       *net.TCPConn
	Id         ksuid.KSUID
	closeable  bool
}

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
		log.Println(err)
		return nil
	} else {
		id = _id
	}
	switch socketStream.Type {
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
				if err := s.forData(connector, &requestMsg); err != nil {
					log.Println(err)
					s.tellClosed(id)
					s.Close(id) // also closed= tcp connection if it exists
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

	if _, err := connector.sendBuffer.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		return err
	}
	log.Println("info", "connected to:", addr)

	defer connector.sendBuffer.Flush(websocket.TextMessage, id, &(s.ConcurrentWebSocket))
	stopper := make(chan bool)
	go func() {
		// defer setDone()
		tick := time.NewTicker(time.Millisecond * time.Duration(60))
		//for range time.Tick(120 * time.Millisecond){}
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if _, err := connector.sendBuffer.Flush(websocket.TextMessage, id, &(s.ConcurrentWebSocket)); err != nil {
					log.Println("Error: error sending data via webSocket:", err)
					return
				}
			case <-stopper:
				return
			}
		}
	}()

	if _, err := io.Copy(&connector.sendBuffer, connector.Conn); err != nil {
		stopper <- true
		return err
	}

	stopper <- true
	return nil
}

func (s *ServerWS) forData(connector *Connector, message *RequestMessage) error {
	// copy data
	if decodeBytes, err := base64.StdEncoding.DecodeString(message.DataBase64); err != nil {
		log.Println("bash64 decode error,", err)
		return err
	} else {
		if _, err := connector.Conn.Write(decodeBytes); err != nil {
			return err
		}
	}
	return nil
}
