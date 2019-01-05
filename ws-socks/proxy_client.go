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
}

// can't do large compute or communication here
func (p *Proxy) DispatchData(data *ProxyData) error {
	// decode base64
	if decodeBytes, err := base64.StdEncoding.DecodeString(data.DataBase64); err != nil { // todo ignore error
		log.Println("bash64 decode error,", err)
		return nil // skip error
	} else {
		if _, err := p.Conn.Write(decodeBytes); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proxy) Close() {
	p.Conn.Close()
}

// handel socket dial results processing
func (p *Proxy) Serve(wsc *WebSocketClient, addr string) error {
	log.Println("dialing to", addr)
	defer log.Println("closing", addr)

	addrSend := WebSocketMessage2{Type: WsTpEst, Id: p.Id.String(), Data: ProxyMessage{Addr: addr}}
	if err := wsc.WriteWSJSON(&addrSend); err != nil {
		log.Println(err)
		return err
	}
	log.Println("connected to", addr)

	done := make(chan bool)
	setDone := func() {
		done <- true
	}

	var buffer Base64WSBufferWriter
	go func() {
		defer setDone()
		buff := make([]byte, 32*1024)
		for {
			if _, err := io.CopyBuffer(&buffer, p.Conn, buff); err != nil { // copy data to buffer
				log.Println("io copy error,", err)
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Microsecond * time.Duration(10))
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return nil // todo error back
		case <-ticker.C:
			err := buffer.Flush(websocket.TextMessage, p.Id, &(wsc.ConcurrentWebSocket))
			if err != nil {
				log.Println("write:", err) // todo use of closed network connection
			}
		}
	}
}
