package wss

import (
	"github.com/genshen/wssocks/wss/ticker"
	log "github.com/sirupsen/logrus"
	"net"
)

// listen on local address:port and forward socks5 requests to wssocks server.
func ListenAndServe(wsc *WebSocketClient, tick *ticker.Ticker, address string) error {
	s, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	log.WithField("local address", address).Info("listening on local address for incoming proxy request.")
	var client Client
	for {
		c, err := s.Accept()
		if err != nil {
			return nil
		}
		log.WithField("size", wsc.ConnSize()).Trace("size of proxy connection changed.")

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
