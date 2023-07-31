package wss

import (
	"context"
	"errors"
	"fmt"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
)

var StoppedError = errors.New("listener stopped")

// client part of wssocks
type Client struct {
	tcpl    *net.TCPListener
	stop    chan interface{}
	closed  bool
	wgClose sync.WaitGroup // wait for closing
}

func NewClient() *Client {
	var client Client
	client.closed = false
	client.stop = make(chan interface{})
	return &client
}

// parse target address and proxy type, and response to socks5/https client
func (client *Client) Reply(conn net.Conn, enableHttp bool) ([]byte, int, string, error) {
	var buffer [1024]byte
	var addr string
	var proxyType int

	n, err := conn.Read(buffer[:])
	if err != nil {
		return nil, 0, "", err
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
		return nil, 0, "", errors.New("only socks5 or http(s) proxy")
	}

	// set address and type
	if proxyAddr, err := matchedInstance.ParseHeader(conn, buffer[:n]); err != nil {
		return nil, 0, "", err
	} else {
		proxyType = matchedInstance.ProxyType()
		addr = proxyAddr
	}
	// set data sent in establish step.
	if firstSendData, err := matchedInstance.EstablishData(buffer[:n]); err != nil {
		return nil, 0, "", err
	} else {
		// firstSendData can be nil, which means there is no data to be send during connection establishing.
		return firstSendData, proxyType, addr, nil
	}
}

// listen on local address:port and forward socks5 requests to wssocks server.
func (client *Client) ListenAndServe(record *ConnRecord, wsc *WebSocketClient, address string, enableHttp bool, onConnected func()) error {
	netListener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	tcpl, ok := (netListener).(*net.TCPListener)
	if !ok {
		return errors.New("not a tcp listener")
	}
	client.tcpl = tcpl

	onConnected()
	for {
		// check stop first
		select {
		case <-client.stop:
			return StoppedError
		default:
			// if the channel is still open, continue as normal
		}

		c, err := tcpl.Accept()
		if err != nil {
			return fmt.Errorf("tcp accept error: %w", err)
		}

		go func() {
			conn := c.(*net.TCPConn)
			// defer c.Close()
			defer conn.Close()
			// In reply, we can get proxy type, target address and first send data.
			firstSendData, proxyType, addr, err := client.Reply(conn, enableHttp)
			if err != nil {
				log.Error("reply error: ", err)
			}
			client.wgClose.Add(1)
			defer client.wgClose.Done()

			// update connection record
			record.Update(ConnStatus{IsNew: true, Address: addr, Type: proxyType})
			defer record.Update(ConnStatus{IsNew: false, Address: addr, Type: proxyType})

			// on connection established, copy data now.
			if err := client.transData(wsc, conn, firstSendData, proxyType, addr); err != nil {
				log.Error("trans error: ", err)
			}
		}()
	}
}

func (client *Client) transData(wsc *WebSocketClient, conn *net.TCPConn, firstSendData []byte, proxyType int, addr string) error {
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
        err := wsc.TellClose(proxy.Id)
        if err != nil {
			log.Error("close error", err)
		}
		return err
	}

	// trans incoming data from proxy client application.
	ctx, cancel := context.WithCancel(context.Background())
	writer := NewWebSocketWriterWithMutex(&wsc.ConcurrentWebSocket, proxy.Id, ctx)
	go func() {
		_, err := io.Copy(writer, conn)
		if err != nil {
			log.Error("write error: ", err)
		}
        done <- Done{true, err}
	}()
	defer writer.CloseWsWriter(cancel) // cancel data writing

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
}

// Close stops listening on the TCP address,
// But the active links are not closed and wait them to finish.
func (client *Client) Close(wait bool) error {
	if client.closed {
		return nil
	}
	close(client.stop)
	client.closed = true
	err := client.tcpl.Close()
	if wait {
		client.wgClose.Wait() // wait the active connection to finish
	}
	return err
}
