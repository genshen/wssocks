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
	"time"
)

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
}

type ServerWS struct {
	ConcurrentWebSocket
	conns map[ksuid.KSUID]*Connector // fixme fix error: concurrent map writes
}

func (s *ServerWS) NewConn(id ksuid.KSUID, conn *net.TCPConn) *Connector {
	connector := Connector{Id: id, Conn: conn}
	s.conns[id] = &connector
	return &connector
}

func (s *ServerWS) Close(id ksuid.KSUID) error {
	if connector, ok := s.conns[id]; ok {
		if err := connector.Conn.Close(); err != nil {
			delete(s.conns, id)
			return err
		} else {
			delete(s.conns, id)
		}
	}
	return nil
}

// in this case, one ws only handle one proxy.
func (s *ServerWS) dispatchMessage(data []byte) error {
	var socketData json.RawMessage
	socketStream := WebSocketMessage2{
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

	case WsTpEst: // establish
		var proxyMsg ProxyMessage
		if err := json.Unmarshal(socketData, &proxyMsg); err != nil {
			return nil
		} else {
			go s.establish(id, proxyMsg.Addr) // todo error handle
		}
	case WsTpData:
		var requestMsg RequestMessage
		if err := json.Unmarshal(socketData, &requestMsg); err != nil {
			return nil
		}

		if connector, ok := s.conns[id]; ok {
			go s.forData(connector, &requestMsg)
		}
		return nil
	}
	return nil
}

func (s *ServerWS) establish(id ksuid.KSUID, addr string) error {
	log.Println("info", "proxy to:", addr)
	tcpConn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		log.Println(err)
		return nil
	}

	connector := s.NewConn(id, tcpConn.(*net.TCPConn))
	defer s.Close(id) // also close tcp connection, todo error

	if _, err := connector.sendBuffer.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		return err
	}
	log.Println("info", "connected to:", addr)

	stopper := make(chan bool)
	go func() {
		// defer setDone()
		tick := time.NewTicker(time.Millisecond * time.Duration(10))
		//for range time.Tick(120 * time.Millisecond){}
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if err := connector.sendBuffer.Flush(websocket.TextMessage, id, &(s.ConcurrentWebSocket)); err != nil {
					log.Println("Error: error sending data via webSocket:", err)
					return
				}
			case <-stopper:
				return
			}
		}
	}()

	if _, err := io.Copy(&connector.sendBuffer, connector.Conn); err != nil {
		return nil
	}
	stopper <- true
	return nil
}

func (s *ServerWS) forData(connector *Connector, message *RequestMessage) error {
	log.Println("send data from client to remote")
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
