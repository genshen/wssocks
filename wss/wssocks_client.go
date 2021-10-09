package wss

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/genshen/wssocks/pipe"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

var StoppedError = errors.New("listener stopped")

var clientQueueHub *pipe.QueueHub
var clientLinkHub *pipe.LinkHub

func init() {
	clientQueueHub = pipe.NewQueueHub()
	clientLinkHub = pipe.NewLinkHub()
}

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
func (client *Client) Reply(conn net.Conn) ([]byte, int, string, error) {
	var buffer [1024]byte
	var addr string
	var proxyType int

	n, err := conn.Read(buffer[:])
	if err != nil {
		return nil, 0, "", err
	}

	// 去掉socks5之外的支持
	// select a matched proxy type
	instances := []ProxyInterface{&Socks5Client{}}
	var matchedInstance ProxyInterface = nil
	for _, proxyInstance := range instances {
		if proxyInstance.Trigger(buffer[:n]) {
			matchedInstance = proxyInstance
			break
		}
	}

	if matchedInstance == nil {
		return nil, 0, "", errors.New("only socks5 proxy")
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
func (client *Client) ListenAndServe(record *ConnRecord, wsc *WebSocketClient, wsc2 *WebSocketClient, address string, onConnected func()) error {
	netListener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	tcpl, ok := (netListener).(*net.TCPListener)
	if !ok {
		return errors.New("not a tcp listener")
	}
	client.tcpl = tcpl

	// 在client刚启动，连上ws server以后要做的事
	onConnected()
	for {
		// 先检查stop 如果已经被close 不再接收新请求
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
			firstSendData, proxyType, addr, err := client.Reply(conn)
			if err != nil {
				log.Error("reply error: ", err)
			}
			// 在client.Close中使用wait等待
			client.wgClose.Add(1)
			defer client.wgClose.Done()

			switch proxyType {
			case ProxyTypeSocks5:
				conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			case ProxyTypeHttps:
				conn.Write([]byte("HTTP/1.0 200 Connection Established\r\nProxy-agent: wssocks\r\n\r\n"))
			}

			// update connection record
			record.Update(ConnStatus{IsNew: true, Address: addr, Type: proxyType})
			defer record.Update(ConnStatus{IsNew: false, Address: addr, Type: proxyType})

			// 转换数据，看怎么分配wsc和wsc2
			// on connection established, copy data now.
			if err := client.transData(wsc, wsc2, conn, firstSendData, addr); err != nil {
				log.Error("trans error: ", err)
			}
		}()
	}
}

// 传输数据
func (client *Client) transData(wsc *WebSocketClient, wsc2 *WebSocketClient, conn *net.TCPConn, firstSendData []byte, addr string) error {
	// create a with proxy with callback func
	proxy := wsc.NewProxy(func(id ksuid.KSUID, data ServerData) { //ondata 接收数据回调
		if data.Tag == TagData {
			clientLinkHub.Write(id, data.Data)
		}
	}, func(id ksuid.KSUID, tell bool) { //onclosed
	}, func(id ksuid.KSUID, err error) { //onerror
	})

	// 第二条线
	proxy2 := wsc2.NewProxy(func(id ksuid.KSUID, data ServerData) { //ondata 接收数据回调
		if data.Tag == TagData {
			clientLinkHub.Write(id, data.Data)
		}
	}, func(id ksuid.KSUID, tell bool) { //onclosed
	}, func(id ksuid.KSUID, err error) { //onerror
	})
	defer func() {
		wsc.RemoveProxy(proxy.Id)
		wsc2.RemoveProxy(proxy2.Id)
	}()

	// 给主链接发送顺序
	sorted := []ksuid.KSUID{proxy.Id, proxy2.Id}

	// 告知服务端目标地址和协议，还有首次发送的数据包, 额外告知有几路以及顺序如何
	if err := proxy.Establish(wsc, firstSendData, addr, sorted); err != nil {
		return err
	}
	// 第二条线路不需要Establish因为不用和目标机器连接

	// 让各自连接准备，对方收到后与总连接数对比决定是否开始向外转发
	proxy.SayID(wsc, proxy.Id)
	proxy2.SayID(wsc2, proxy.Id) //都发送主id

	// trans incoming data from proxy client application.
	ctx, cancel := context.WithCancel(context.Background())
	writer := NewWebSocketWriterWithMutex(&wsc.ConcurrentWebSocket, proxy.Id, ctx)

	ctx2, cancel2 := context.WithCancel(context.Background())
	writer2 := NewWebSocketWriterWithMutex(&wsc2.ConcurrentWebSocket, proxy2.Id, ctx2)
	defer func() {
		writer.CloseWsWriter(cancel)  // cancel data writing
		writer.CloseWsWriter(cancel2) // cancel data writing
	}()

	//发送数据
	clientQueueHub.Add(proxy.Id, proxy.Id, writer)
	clientQueueHub.Add(proxy.Id, proxy2.Id, writer2)
	qq := clientQueueHub.Get(proxy.Id)
	// 设置发送顺序
	qq.SetSort(sorted)
	go qq.Send()
	defer clientQueueHub.Remove(proxy.Id)

	go func() {
		_, err := pipe.CopyBuffer(qq, conn) //io.Copy(qq, conn)
		if err != nil {
			log.Error("write error: ", err)
		}
	}()

	//接收数据
	clientLinkHub.Add(proxy.Id, proxy.Id)
	clientLinkHub.Add(proxy2.Id, proxy.Id)
	defer clientLinkHub.RemoveAll(proxy.Id)

	oo := clientLinkHub.Get(proxy.Id)
	// 设置接收的数据发送到哪
	oo.SetConn(conn)
	oo.SetSort(sorted)
	go oo.Send(clientLinkHub)

	fmt.Println("wait")
	oo.Wait()
	fmt.Println("done")
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
