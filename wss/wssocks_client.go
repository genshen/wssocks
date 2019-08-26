package wss

import (
	"errors"
	"github.com/genshen/wssocks/wss/term_view"
	log "github.com/sirupsen/logrus"
	"net"
)

// client part of wssocks
type Client struct {
}

// response to socks5 client and start to exchange data between socks5 client and
func (client *Client) Reply(conn net.Conn, enableHttp bool,
	onDial func(conn *net.TCPConn, firstSendData []byte, proxyType int, addr string) error) error {
	defer conn.Close()
	var buffer [1024]byte
	var firstSendData []byte = nil
	var addr string
	var proxyType int

	n, err := conn.Read(buffer[:])
	if err != nil {
		return err
	}

	// select a matched proxy type
	instances := []ProxyInterface{&Socks5Client{}}
	if enableHttp { // if http and https proxy is enabled.
		instances = append(instances, &HttpClient{}, &HttpsClient{})
	}
	var matchedInstance ProxyInterface = nil
	for _, proxyInstance := range instances {
		if proxyInstance.Trigger(buffer[:n]) {
			matchedInstance = proxyInstance
			break
		}
	}

	if matchedInstance == nil {
		return errors.New("only socks5 or http(s) proxy")
	}

	// set address and type
	if proxyAddr, err := matchedInstance.ParseHeader(conn, buffer[:n]); err != nil {
		return err
	} else {
		proxyType = matchedInstance.ProxyType()
		addr = proxyAddr
	}
	// set data sent in establish step.
	if newBuffer, err := matchedInstance.EstablishData(buffer[:n]); err != nil {
		return err
	} else {
		firstSendData = newBuffer
	}

	//  dial to target.
	// firstSendData can be nil, which means there is no data to be send during connection establishing.
	if err := onDial(conn.(*net.TCPConn), firstSendData, proxyType, addr); err != nil {
		return err
	}
	return nil
}

// listen on local address:port and forward socks5 requests to wssocks server.
func ListenAndServe(wsc *WebSocketClient, address string, enableHttp bool) error {
	s, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	log.WithField("local address", address).Info("listening on local address for incoming proxy request.")

	plog := term_view.NewPLog()
	log.SetOutput(plog) // change log stdout to plog

	var client Client
	for {
		c, err := s.Accept()
		if err != nil {
			return err
		}

		go func() {
			err := client.Reply(c, enableHttp, func(conn *net.TCPConn, firstSendData []byte, proxyType int, addr string) error {
				proxy := wsc.NewProxy(conn)
				proxy.Serve(plog, wsc, firstSendData, proxyType, addr)
				wsc.TellClose(proxy.Id)
				return nil // todo error
			})
			if err != nil {
				log.Error(err)
			}
		}()
	}
}
