package ws_socks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/genshen/ws-socks/ws-socks/ticker"
	"log"
	"net"
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

	wsc := WebSocketClient{}
	wsc.Connect(client.Config.ServerAddr)
	log.Println("connected")
	// todo chan for wsc and tcp accept
	defer wsc.WSClose()
	go wsc.ListenIncomeMsg()

	tick := ticker.NewTicker()
	tick.Start()
	defer tick.Stop()

	for {
		log.Println("size of connector:", wsc.ConnSize())
		c, err := s.Accept()
		if err != nil {
			log.Panic(err)
		}
		go client.proxy(c, func(conn *net.TCPConn, addr string) error {
			proxy := wsc.NewProxy(conn)
			proxy.Serve(&wsc, tick, addr)
			wsc.TellClose(proxy.Id)
			return nil // todo error
		})
	}
}

func (client *Client) proxy(conn net.Conn, onDial func(conn *net.TCPConn, addr string) error) {
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
			// ipv4 address
			type sockIP struct {
				A, B, C, D byte
				PORT       uint16
			}
			sip := sockIP{}
			if err := binary.Read(bytes.NewReader(b[4:n]), binary.BigEndian, &sip); err != nil {
				log.Println("error request for ipv4,", err)
				return
			}
			addr = fmt.Sprintf("%d.%d.%d.%d:%d", sip.A, sip.B, sip.C, sip.D, sip.PORT)
		case 0x03:
			// domain
			host := string(b[5 : n-2])
			var port uint16
			err = binary.Read(bytes.NewReader(b[n-2:n]), binary.BigEndian, &port)
			if err != nil {
				log.Println(err)
				return
			}
			addr = fmt.Sprintf("%s:%d", host, port)
		}

		if err := onDial(conn.(*net.TCPConn), addr); err != nil {
			log.Println(err)
		}
		//client.localDail(conn.(*net.TCPConn), addr)
	}
}
