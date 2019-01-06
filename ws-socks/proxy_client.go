package ws_socks

import (
	"encoding/base64"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"io"
	"log"
	"net"
	"time"
)

type Proxy struct {
	Conn       *net.TCPConn
	Id         ksuid.KSUID
	SendBuffer Base64WSBufferWriter
	isClosed   bool
}

// can't do large compute or communication here
func (p *Proxy) DispatchData(data *ProxyData) error {
	// decode base64
	if decodeBytes, err := base64.StdEncoding.DecodeString(data.DataBase64); err != nil { // todo ignore error
		log.Println("bash64 decode error,", err)
		return err // skip error
	} else {
		if _, err := p.Conn.Write(decodeBytes); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proxy) Close() {
	if p.isClosed {
		return
	}
	p.Conn.Close()
	p.isClosed = true
}

// handel socket dial results processing
func (p *Proxy) Serve(wsc *WebSocketClient, addr string) error {
	log.Println("dialing to", addr)
	defer log.Println("closing", addr)
	defer wsc.Close(p.Id)

	addrSend := WebSocketMessage2{Type: WsTpEst, Id: p.Id.String(), Data: ProxyMessage{Addr: addr}}
	if err := wsc.WriteWSJSON(&addrSend); err != nil {
		log.Println(err)
		return err
	}
	log.Println("connected to", addr)

	done := make(chan bool)
	var buffer Base64WSBufferWriter
	defer buffer.Flush(websocket.TextMessage, p.Id, &(wsc.ConcurrentWebSocket))

	go func() {
		ticker := time.NewTicker(time.Microsecond * time.Duration(60))
		defer ticker.Stop()
		for {
			select {
			case <-done:
				log.Println("done")
				return // todo error back
			case <-ticker.C:
				_, err := buffer.Flush(websocket.TextMessage, p.Id, &(wsc.ConcurrentWebSocket))
				if err != nil {
					log.Println("write:", err) // todo use of closed network connection
				}
			}
		}
	}()

	if _, err := io.Copy(&buffer, p.Conn); err != nil { // copy data to buffer
		log.Println("io copy error,", err)
		done <- true
		return nil
	}
	done <- true
	return nil
}
