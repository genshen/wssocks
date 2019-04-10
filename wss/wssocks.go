package wss

import (
	"github.com/genshen/wssocks/wss/ticker"
	"log"
	"net"
)

// listen on local address:port and forward socks5 requests to wssocks server.
func ListenAndServe(wsc *WebSocketClient, tick *ticker.Ticker, address string) error {
	s, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	var client Client
	for {
		log.Println("size of connector:", wsc.ConnSize())
		c, err := s.Accept()
		if err != nil {
			return nil
		}
		go func() {
			err := client.Reply(c, func(conn *net.TCPConn, addr string) error {
				proxy := wsc.NewProxy(conn)
				proxy.Serve(wsc, tick, addr)
				wsc.TellClose(proxy.Id)
				return nil // todo error
			})
			if err != nil {
				log.Println(err)
			}
		}()
	}
}
