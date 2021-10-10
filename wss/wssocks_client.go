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
func (client *Client) ListenAndServe(record *ConnRecord, wsc []*WebSocketClient, address string, onConnected func()) error {
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

			// 更新输出区域的连接数据
			record.Update(ConnStatus{IsNew: true, Address: addr, Type: proxyType})
			defer record.Update(ConnStatus{IsNew: false, Address: addr, Type: proxyType})

			// 传输数据
			// on connection established, copy data now.
			if err := client.transData(wsc, conn, firstSendData, addr); err != nil {
				log.Error("trans error: ", err)
			}
		}()
	}
}

// 传输数据
func (client *Client) transData(wsc []*WebSocketClient, conn *net.TCPConn, firstSendData []byte, addr string) error {
	var masterProxy *ProxyClient
	var masterID ksuid.KSUID
	var sorted []ksuid.KSUID
	for i, w := range wsc {
		// create a with proxy with callback func
		p := w.NewProxy(func(id ksuid.KSUID, data ServerData) { //ondata 接收数据回调
			if data.Tag == TagData {
				clientLinkHub.Write(id, data.Data)
			} else if data.Tag == TagEOF {
				//fmt.Println("client receive eof")
				clientLinkHub.Get(id).WriteEOF()
			}
		}, func(id ksuid.KSUID, tell bool) { //onclosed
			//服务器出错让关闭，关闭双向的通道
			clientQueueHub.Remove(id)
			clientLinkHub.RemoveAll(id)
		}, func(id ksuid.KSUID, err error) { //onerror
		})
		defer w.RemoveProxy(p.Id)

		// 第一个做为主id
		if i == 0 {
			masterID = p.Id
			masterProxy = p
		}
		// 给主链接发送的顺序
		sorted = append(sorted, p.Id)
		// 让各自连接准备，对方收到后与总连接数对比决定是否开始向外转发
		// 最好放在Establish前发送，这样Establish数据得到进行setSort时map一定存在
		p.SayID(w, masterID)

		// trans incoming data from proxy client application.
		ctx, cancel := context.WithCancel(context.Background())
		writer := NewWebSocketWriterWithMutex(&w.ConcurrentWebSocket, p.Id, ctx)
		defer writer.CloseWsWriter(cancel)

		clientQueueHub.Add(masterID, p.Id, writer)
		clientLinkHub.Add(p.Id, masterID)
	}

	defer func() {
		clientQueueHub.Remove(masterID)
		clientLinkHub.RemoveAll(masterID)
	}()

	// 告知服务端目标地址和协议，还有首次发送的数据包, 额外告知有几路以及顺序如何
	// 第二到N条线路不需要Establish因为不用和目标机器连接
	if err := masterProxy.Establish(wsc[0], firstSendData, addr, sorted); err != nil {
		return err
	}

	//发送数据
	qq := clientQueueHub.Get(masterID)
	// 设置发送顺序
	qq.SetSort(sorted)
	go qq.Send()

	go func() {
		_, err := pipe.CopyBuffer(qq, conn) //io.Copy(qq, conn)
		if err != nil {
			log.Error("write error: ", err)
		}
	}()

	//接收数据
	oo := clientLinkHub.Get(masterID)
	// 设置接收的数据发送到哪
	oo.SetConn(conn)
	oo.SetSort(sorted)
	go oo.Send(clientLinkHub)

	//fmt.Println(clientLinkHub.Len(), clientQueueHub.Len())
	//time.Sleep(time.Minute)
	//fmt.Println("wait")
	oo.Wait()
	//fmt.Println("done")
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
