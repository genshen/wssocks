package wss

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/genshen/wssocks/pipe"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

var serverQueueHub *pipe.QueueHub
var serverLinkHub *pipe.LinkHub

func init() {
	serverQueueHub = pipe.NewQueueHub()
	serverLinkHub = pipe.NewLinkHub()
}

type Connector struct {
	Conn io.ReadWriteCloser
}

// interface of establishing proxy connection with target
type ProxyEstablish interface {
	establish(hub *Hub, id ksuid.KSUID, addr string, data []byte) error

	// close connection
	Close(id ksuid.KSUID) error
}

type ClientData ServerData

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
	// debug
	//if socketStream.Type != WsTpBeats {
	//	fmt.Println("dispatch", id, socketStream.Type)
	//}

	switch socketStream.Type {
	case WsTpBeats: // heart beats
	case WsTpClose: // closed by client
		// 应该永远走不到这里，目前不需要客户端传close给服务端
	case WsTpHi:
		var masterID ksuid.KSUID
		if err := json.Unmarshal(socketData, &masterID); err != nil {
			return err
		}
		writer := NewWebSocketWriter(&hub.ConcurrentWebSocket, id, context.Background())

		serverQueueHub.Add(masterID, id, writer)
		serverLinkHub.Add(id, masterID)
		//fmt.Println("get client say", id, masterID)
	case WsTpEst: // establish 收到连接请求
		var proxyEstMsg ProxyEstMessage
		if err := json.Unmarshal(socketData, &proxyEstMsg); err != nil {
			return err
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
		//fmt.Println("est", id, proxyEstMsg.Sorted)
		serverQueueHub.SetSort(id, proxyEstMsg.Sorted)
		serverLinkHub.SetSort(id, proxyEstMsg.Sorted)
		// 与外面建立连接，并把外面返回的数据放回websocket
		go establishProxy(hub, ProxyRegister{id, proxyEstMsg.Addr, estData})
	case WsTpData: //从websocket收到数据发送到外面
		var requestMsg ProxyData
		if err := json.Unmarshal(socketData, &requestMsg); err != nil {
			fmt.Println("json", err)
			return err
		}

		if requestMsg.Tag == TagEOF { //设置收到io.EOF结束符
			//fmt.Println("server receive eof")
			serverLinkHub.Get(id).WriteEOF()
			return nil
		}
		if decodeBytes, err := base64.StdEncoding.DecodeString(requestMsg.DataBase64); err != nil {
			log.Error("base64 decode error,", err)
			return err
		} else {
			//fmt.Println("bytes", id, len(decodeBytes), string(decodeBytes))
			// 传输数据
			serverLinkHub.Write(id, decodeBytes)
			return nil
		}
	}
	return nil
}

func establishProxy(hub *Hub, proxyMeta ProxyRegister) {
	var e ProxyEstablish
	e = &DefaultProxyEst{}

	err := e.establish(hub, proxyMeta.id, proxyMeta.addr, proxyMeta.withData)
	if err == nil {
		// 当连接后端服务器失败时，让客户端断开
		hub.tellClosed(proxyMeta.id) // tell client to close connection.
	}
}

// interface implementation for socks5 and https proxy.
type DefaultProxyEst struct {
}

func (e *DefaultProxyEst) Close(id ksuid.KSUID) error {
	serverLinkHub.RemoveAll(id)
	serverQueueHub.Remove(id)
	return nil
}

// data: data send in establish step (can be nil).
func (e *DefaultProxyEst) establish(hub *Hub, id ksuid.KSUID, addr string, data []byte) error {
	conn, err := net.DialTimeout("tcp", addr, time.Second*8) // todo config timeout
	if err != nil {
		return err
	}
	//收集请求发送出去
	serverLinkHub.TrySend(id, conn.(*net.TCPConn))
	defer func() {
		conn.Close()
		e.Close(id)
	}()

	// todo check exists
	hub.addNewProxy(&ProxyServer{Id: id, ProxyIns: e})
	defer hub.RemoveProxy(id)

	serverQueueHub.TrySend(id)
	writer := serverQueueHub.Get(id)
	go func() {
		// 从外面往回接收数据
		_, err := pipe.CopyBuffer(writer, conn.(*net.TCPConn))
		if err != nil {
			log.Error("copy error,", err)
		}
	}()

	//fmt.Println(serverLinkHub.Len(), serverQueueHub.Len())
	//time.Sleep(time.Minute)
	//fmt.Println("wait")
	writer.Wait()
	//fmt.Println("done")
	// s.RemoveProxy(proxy.Id)
	// tellClosed is called outside this func.
	return nil
}
