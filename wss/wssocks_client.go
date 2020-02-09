package wss

import (
	"errors"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"io"
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
		instances = append(instances, &HttpsClient{})
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
func ListenAndServe(record *ConnRecord, wsc *WebSocketClient, address string, enableHttp bool, onConnected func()) error {
	s, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	onConnected()
	var client Client
	for {
		c, err := s.Accept()
		if err != nil {
			return err
		}

		go func() {
			err := client.Reply(c, enableHttp, func(conn *net.TCPConn, firstSendData []byte, proxyType int, addr string) error {
				defer conn.Close()

				record.Update(ConnStatus{IsNew: true, Address: addr, Type: proxyType})
				defer record.Update(ConnStatus{IsNew: false, Address: addr, Type: proxyType})

				type Done struct {
					tell bool
					err  error
				}
				done := make(chan Done, 2)
				// defer close(done)

				// create a with proxy with callback func
				proxy := wsc.NewProxy(func(id ksuid.KSUID, data ServerData) {
					if _, err := conn.Write(data.Data); err != nil {
						done <- Done{true, err}
					}
				}, func(id ksuid.KSUID, tell bool) {
					done <- Done{tell, nil}
				}, func(id ksuid.KSUID, err error) {
					if err != nil {
						done <- Done{true, err}
					}
				})

				// tell server to establish connection
				if err := proxy.Establish(wsc, firstSendData, proxyType, addr); err != nil {
					wsc.RemoveProxy(proxy.Id)
					if err := wsc.TellClose(proxy.Id); err != nil {
						log.Error("close error", err)
					}
					return err
				}

				// trans incoming data from proxy client application.
				go func() {
					writer := WebSocketWriter{WSC: &wsc.ConcurrentWebSocket, Id: proxy.Id}
					if _, err := io.Copy(&writer, conn); err != nil {
						log.Error("write error:", err)
					}
					done <- Done{true, nil}
				}()

				d := <-done
				wsc.RemoveProxy(proxy.Id)
				if d.tell {
					if err := wsc.TellClose(proxy.Id); err != nil {
						return err
					}
				}
				if d.err != nil {
					return d.err
				}
				return nil
			})
			if err != nil {
				log.Error(err)
			}
		}()
	}
}
