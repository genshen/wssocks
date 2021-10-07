package wss

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

var serverQueueHub *queueHub

var outQueueHub *queueHub2

func init() {
	serverQueueHub = NewQueueHub()
	outQueueHub = NewQueueHub2()
}

type Connector struct {
	Conn io.ReadWriteCloser
}

// interface of establishing proxy connection with target
type ProxyEstablish interface {
	establish(hub *Hub, id ksuid.KSUID, proxyType int, addr string, data []byte) error

	// data from client todo data with type
	onData(id ksuid.KSUID, data ClientData) error

	// close connection
	// tell: whether to send close message to proxy client
	Close(tell bool) error
}

type ClientData ServerData

var ConnCloseByClient = errors.New("conn closed by client")

func dispatchMessage(hub *Hub, msgType websocket.MessageType, data []byte, config WebsocksServerConfig) error {
	if msgType == websocket.MessageText {
		return dispatchDataMessage(hub, data, config)
	}
	return nil
}

func dispatchDataMessage(hub *Hub, data []byte, config WebsocksServerConfig) error {
	var socketData json.RawMessage
	socketStream := WebSocketMessage{
		Data: &socketData,
	}
	if err := json.Unmarshal(data, &socketStream); err != nil {
		fmt.Println(err)
		return err
	}

	// parsing id
	id, err := ksuid.Parse(socketStream.Id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	//if socketStream.Type != WsTpBeats {
	//	fmt.Println("dispatch", id, socketStream.Type)
	//}

	switch socketStream.Type {
	case WsTpBeats: // heart beats
	case WsTpClose: // closed by client
		return hub.CloseProxyConn(id)
	case WsTpHi:
		var masterId string
		if err := json.Unmarshal(socketData, &masterId); err != nil {
			return err
		}
		writer := NewWebSocketWriter(&hub.ConcurrentWebSocket, id, context.Background())
		masterKSUID, err := ksuid.Parse(masterId)
		if err != nil {
			return err
		}
		serverQueueHub.addWriter(masterKSUID, id, writer)
		serverQueueHub.Incre(masterKSUID)
		serverQueueHub.TrySend(masterKSUID)

		outQueueHub.addBufQueue(id, masterKSUID)
		outQueueHub.Incre(masterKSUID)
		outQueueHub.TrySend(masterKSUID, nil)
		outQueueHub.SetMap(id, masterKSUID)
		//fmt.Println("get client say", id)
	case WsTpEst: // establish 收到连接请求
		var proxyEstMsg ProxyEstMessage
		if err := json.Unmarshal(socketData, &proxyEstMsg); err != nil {
			return err
		}
		// check proxy type support.
		if (proxyEstMsg.Type == ProxyTypeHttp || proxyEstMsg.Type == ProxyTypeHttps) && !config.EnableHttp {
			hub.tellClosed(id) // tell client to close connection.
			return errors.New("http(s) proxy is not support in server side")
		}

		var estData []byte = nil
		if proxyEstMsg.WithData {
			if decodedBytes, err := base64.StdEncoding.DecodeString(proxyEstMsg.DataBase64); err != nil {
				log.Error("base64 decode error,", err)
				return err
			} else {
				estData = decodedBytes
			}
		}
		serverQueueHub.GetById(id).SetSort(proxyEstMsg.Sorted)
		outQueueHub.GetById(id).SetSort(proxyEstMsg.Sorted)
		// 与外面建立连接，并把外面返回的数据放回websocket
		go establishProxy(hub, ProxyRegister{id, proxyEstMsg.Type, proxyEstMsg.Addr, estData})
	case WsTpData: //从websocket收到数据发送到外面
		var requestMsg ProxyData
		if err := json.Unmarshal(socketData, &requestMsg); err != nil {
			fmt.Println("json", err)
			return err
		}

		if decodeBytes, err := base64.StdEncoding.DecodeString(requestMsg.DataBase64); err != nil {
			log.Error("base64 decode error,", err)
			return err
		} else {
			//fmt.Println("bytes", id, len(decodeBytes), string(decodeBytes))
			// 传输数据
			outQueueHub.GetById(id).setData(decodeBytes)
			return nil
		}

	}
	return nil
}

func establishProxy(hub *Hub, proxyMeta ProxyRegister) {
	var e ProxyEstablish
	e = &DefaultProxyEst{}

	err := e.establish(hub, proxyMeta.id, proxyMeta._type, proxyMeta.addr, proxyMeta.withData)
	if err == nil {
		hub.tellClosed(proxyMeta.id) // tell client to close connection.
	} else if err != ConnCloseByClient {
		log.Error(err) // todo error handle better way
		hub.tellClosed(proxyMeta.id)
	}
	return
	//	log.WithField("size", s.GetConnectorSize()).Trace("connection size changed.")
}

// data type used in DefaultProxyEst to pass data to channel
type ChanDone struct {
	tell bool
	err  error
}

// interface implementation for socks5 and https proxy.
type DefaultProxyEst struct {
	done chan ChanDone
	//tcpConn net.Conn
}

func (e *DefaultProxyEst) onData(id ksuid.KSUID, data ClientData) error {
	return nil
}

func (e *DefaultProxyEst) Close(tell bool) error {
	e.done <- ChanDone{tell, ConnCloseByClient}
	return nil // todo error
}

// data: data send in establish step (can be nil).
func (e *DefaultProxyEst) establish(hub *Hub, id ksuid.KSUID, proxyType int, addr string, data []byte) error {
	conn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}
	//收集请求发送出去
	outQueueHub.TrySend(id, conn)
	defer conn.Close()

	e.done = make(chan ChanDone, 2)
	//defer close(done)

	// todo check exists
	hub.addNewProxy(&ProxyServer{Id: id, ProxyIns: e})
	defer hub.RemoveProxy(id)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	switch proxyType {
	case ProxyTypeSocks5:
		if err := hub.WriteProxyMessage(ctx, id, TagHandshake, []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
			return err
		}
	case ProxyTypeHttps:
		if err := hub.WriteProxyMessage(ctx, id, TagHandshake, []byte("HTTP/1.0 200 Connection Established\r\nProxy-agent: wssocks\r\n\r\n")); err != nil {
			return err
		}
	}

	serverQueueHub.TrySend(id)
	writer := serverQueueHub.GetById(id)
	go func() {
		// 往回传
		_, err := copyBuffer(writer, conn.(*net.TCPConn))
		if err != nil {
			log.Error("copy error,", err)
			e.done <- ChanDone{true, err}
		}
		e.done <- ChanDone{true, nil}
	}()

	d := <-e.done
	// s.RemoveProxy(proxy.Id)
	// tellClosed is called outside this func.
	return d.err
}
