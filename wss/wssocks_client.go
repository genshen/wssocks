package wss

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

var StoppedError = errors.New("listener stopped")

var clientQueueHub *queueHub
var clientBackHub *queueHub2

func init() {
	clientQueueHub = NewQueueHub()
	clientBackHub = NewQueueHub2()
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
func (client *Client) ListenAndServe(record *ConnRecord, wsc *WebSocketClient, wsc2 *WebSocketClient, address string, enableHttp bool, onConnected func()) error {
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

			// 转换数据，看怎么分配wsc和wsc2
			// on connection established, copy data now.
			if err := client.transData(wsc, wsc2, conn, firstSendData, proxyType, addr); err != nil {
				log.Error("trans error: ", err)
			}
		}()
	}
}

func (client *Client) transData(wsc *WebSocketClient, wsc2 *WebSocketClient, conn *net.TCPConn, firstSendData []byte, proxyType int, addr string) error {
	type Done struct {
		tell bool
		err  error
	}
	done := make(chan Done, 3)
	// defer close(done)

	// create a with proxy with callback func
	proxy := wsc.NewProxy(func(id ksuid.KSUID, data ServerData) { //ondata
		if data.Tag == TagHandshake {
			if _, err := conn.Write(data.Data); err != nil {
				clientBackHub.GetById(id).Close()
			}
		} else {
			clientBackHub.GetById(id).setData(data.Data)
		}
	}, func(id ksuid.KSUID, tell bool) { //onclosed
		done <- Done{tell, nil}
	}, func(id ksuid.KSUID, err error) { //onerror
		if err != nil {
			done <- Done{true, err}
		}
	})

	// 第二条线
	proxy2 := wsc2.NewProxy(func(id ksuid.KSUID, data ServerData) { //ondata
		if data.Tag == TagHandshake {
			if _, err := conn.Write(data.Data); err != nil {
				clientBackHub.GetById(id).Close()
			}
		} else {
			clientBackHub.GetById(id).setData(data.Data)
		}
	}, func(id ksuid.KSUID, tell bool) { //onclosed
		done <- Done{tell, nil}
	}, func(id ksuid.KSUID, err error) { //onerror
		if err != nil {
			done <- Done{true, err}
		}
	})

	// 让各自连接准备开始
	proxy.SayID(wsc, proxy.Id)
	proxy2.SayID(wsc2, proxy.Id) //都发送主id
	//fmt.Println("client say", proxy.Id, proxy2.Id)

	// 给主链接顺序
	sorted := []ksuid.KSUID{proxy.Id, proxy2.Id}

	// 告知服务端目标地址和协议，还有首次发送的数据包, 额外告知有几路以及顺序如何
	// tell server to establish connection
	//fmt.Println("firstSend", firstSendData)
	if err := proxy.Establish(wsc, firstSendData, proxyType, addr, sorted); err != nil {
		wsc.RemoveProxy(proxy.Id)
		if err := wsc.TellClose(proxy.Id); err != nil {
			log.Error("close error", err)
		}
		return err
	}
	// 第二条线路不需要Establish因为不用和目标机器连接

	// trans incoming data from proxy client application.
	ctx, cancel := context.WithCancel(context.Background())
	writer := NewWebSocketWriterWithMutex(&wsc.ConcurrentWebSocket, proxy.Id, ctx)

	ctx2, cancel2 := context.WithCancel(context.Background())
	writer2 := NewWebSocketWriterWithMutex(&wsc2.ConcurrentWebSocket, proxy2.Id, ctx2)

	clientQueueHub.AddWriter(proxy.Id, proxy.Id, writer)
	clientQueueHub.AddWriter(proxy.Id, proxy2.Id, writer2)
	qq := clientQueueHub.GetById(proxy.Id)
	qq.SetSort(sorted)
	go qq.Send()
	defer qq.Close()

	clientBackHub.addLink(proxy.Id, proxy.Id)
	clientBackHub.addLink(proxy2.Id, proxy.Id)
	oo := clientBackHub.GetById(proxy.Id)
	oo.SetConn(conn)
	oo.SetSort(sorted)
	go func() {
		err := oo.Send(clientBackHub)
		done <- Done{true, err}
	}()

	go func() {
		_, err := copyBuffer(qq, conn) //io.Copy(qq, conn) //client.copyBuffer(qq, conn)
		if err != nil {
			log.Error("write error: ", err)
		}
		done <- Done{true, nil}
	}()
	defer writer.CloseWsWriter(cancel)  // cancel data writing
	defer writer.CloseWsWriter(cancel2) // cancel data writing

	d := <-done
	wsc.RemoveProxy(proxy.Id)
	wsc2.RemoveProxy(proxy2.Id)
	if d.tell { //出错了
		if err := wsc.TellClose(proxy.Id); err != nil {
			return err
		}
		if err := wsc2.TellClose(proxy2.Id); err != nil {
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
