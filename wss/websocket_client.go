package wss

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"net"
	"net/http"
	"sync"
	"time"
)

type WebSocketClient struct {
	ConcurrentWebSocket
	proxies map[ksuid.KSUID]*ProxyClient // all proxies on this websocket.
	proxyMu sync.RWMutex                 // mutex to operate proxies map.
}

// get the connection size
func (wsc *WebSocketClient) ConnSize() int {
	wsc.proxyMu.RLock()
	defer wsc.proxyMu.RUnlock()
	return len(wsc.proxies)
}

// Establish websocket connection.
// And initialize proxies container.
func NewWebSocketClient(dialer *websocket.Dialer, addr string, header http.Header) (*WebSocketClient, error) {
	var wsc WebSocketClient
	ws, _, err := dialer.Dial(addr, header)
	if err != nil {
		return nil, err
	}
	wsc.WsConn = ws
	wsc.proxies = make(map[ksuid.KSUID]*ProxyClient)
	return &wsc, nil
}

// create a new proxy with unique id
func (wsc *WebSocketClient) NewProxy(conn *net.TCPConn) *ProxyClient {
	id := ksuid.New()
	proxy := ProxyClient{Id: id, Conn: conn}
	proxy.isClosed = false

	wsc.proxyMu.Lock()
	defer wsc.proxyMu.Unlock()

	wsc.proxies[id] = &proxy
	return &proxy
}

func (wsc *WebSocketClient) GetProxyById(id ksuid.KSUID) *ProxyClient {
	wsc.proxyMu.RLock()
	defer wsc.proxyMu.RUnlock()
	if proxy, ok := wsc.proxies[id]; ok {
		return proxy
	}
	return nil
}

// tell the remote proxy server to close this connection.
func (wsc *WebSocketClient) TellClose(id ksuid.KSUID) error {
	// send finish flag to client
	finish := WebSocketMessage{
		Id:   id.String(),
		Type: WsTpClose,
		Data: nil,
	}
	if err := wsc.WriteWSJSON(&finish); err != nil {
		return err
	}
	return nil
}

// close current (TCP) connection
func (wsc *WebSocketClient) Close(id ksuid.KSUID) {
	wsc.proxyMu.Lock()
	defer wsc.proxyMu.Unlock()
	if proxy, ok := wsc.proxies[id]; ok {
		proxy.Close()
		delete(wsc.proxies, id)
	}
}

// listen income websocket messages and dispatch to different proxies.
func (wsc *WebSocketClient) ListenIncomeMsg() error {
	for {
		_, data, err := wsc.WsConn.ReadMessage()
		if err != nil {
			// todo close all
			return err // todo close websocket
		}

		var socketData json.RawMessage
		socketStream := WebSocketMessage{
			Data: &socketData,
		}
		if err := json.Unmarshal(data, &socketStream); err != nil {
			continue // todo log
		}
		// find proxy by id
		if ksid, err := ksuid.Parse(socketStream.Id); err != nil {
			continue
		} else {
			if proxy := wsc.GetProxyById(ksid); proxy != nil {
				// now, we known the id and type of incoming data
				switch socketStream.Type {
				case WsTpClose: // remove proxy
					wsc.Close(ksid)
				case WsTpData:
					var proxyData ProxyData
					if err := json.Unmarshal(socketData, &proxyData); err != nil {
						wsc.Close(ksid)
						wsc.TellClose(ksid)
						continue
					}
					if err := proxy.DispatchData(&proxyData); err != nil {
						wsc.Close(ksid)
						wsc.TellClose(ksid)
						continue
					}
				}
			}
		}
	}
}

// start sending heart beat to server.
func (wsc *WebSocketClient) HeartBeat() error {
	t := time.NewTicker(time.Second * 15)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			heartBeats := WebSocketMessage{
				Id:   ksuid.KSUID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}.String(),
				Type: WsTpBeats,
				Data: nil,
			}
			if err := wsc.WriteWSJSON(heartBeats); err != nil {
				return err
			}
		}
	}
	return nil
}
