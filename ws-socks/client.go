package ws_socks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"time"
)

type ClientConfig struct {
	LocalAddr  string // "host:port"
	ServerAddr string
}
type Client struct {
	Config ClientConfig
}

func (client *Client) Start() {
	s, err := net.Listen("tcp", client.Config.LocalAddr)
	if err != nil {
		log.Panic(err)
	}

	for {
		c, err := s.Accept()
		if err != nil {
			log.Panic(err)
		}
		go client.proxy(c)
	}
}

func (client *Client) proxy(conn net.Conn) {
	defer conn.Close()
	var b [1024]byte

	n, err := conn.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}
	var addr string
	//sock5代理
	if b[0] == 0x05 {
		//回应确认代理
		conn.Write([]byte{0x05, 0x00})

		n, err = conn.Read(b[:])
		if err != nil {
			log.Println(err)
			return
		}
		switch b[3] {
		case 0x01:
			//解析代理ip
			type sockIP struct {
				A, B, C, D byte
				PORT       uint16
			}
			sip := sockIP{}
			if err := binary.Read(bytes.NewReader(b[4:n]), binary.BigEndian, &sip); err != nil {
				log.Println("请求解析错误")
				return
			}
			addr = fmt.Sprintf("%d.%d.%d.%d:%d", sip.A, sip.B, sip.C, sip.D, sip.PORT)
		case 0x03:
			//解析代理域名
			host := string(b[5 : n-2])
			var port uint16
			err = binary.Read(bytes.NewReader(b[n-2:n]), binary.BigEndian, &port)
			if err != nil {
				log.Println(err)
				return
			}
			addr = fmt.Sprintf("%s:%d", host, port)
		}

		//client.localDail(conn.(*net.TCPConn), addr)

		// setup websocket
		client.dialWs(conn.(*net.TCPConn), addr)
	}
}


func (client *Client) dialWs(conn *net.TCPConn, addr string) {
	log.Println("dialing to", addr)
	ws, _, err := websocket.DefaultDialer.Dial(client.Config.ServerAddr, nil)
	if err != nil {
		log.Fatal("dial:", err) // todo log level
	}
	defer ws.Close()
	defer log.Println("closing", addr)

	addrSend := WebSocketMessage{Type: WebSocketMessageTypeProxy, Data: ProxyMessage{Addr: addr}}
	if err := ws.WriteJSON(&addrSend); err != nil {
		log.Println(err)
		return
	}
	log.Println("connected to", addr)

	done := make(chan bool, 2)
	setDone := func() {
		done <- true
	}
	go func() {
		defer setDone()
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Println("read data length:", len(message))
			if _, err = conn.Write(message); err != nil {
				log.Println("write error", err)
			}
		}
	}()

	var buffer Base64WSBufferWriter
	go func() {
		defer setDone()
		buff := make([]byte, 32*1024)
		for {
			if _, err := io.CopyBuffer(&buffer, conn, buff); err != nil { // copy data to buffer
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
			return
		case <-ticker.C:
			err := buffer.Flush(websocket.TextMessage, ws)
			if err != nil {
				log.Println("write:", err)
			}
		}
	}
}
