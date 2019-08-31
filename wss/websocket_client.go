package wss

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"io"
	"net/http"
	"sync"
	"time"
)

// WebSocketClient is a collection of proxy clients.
// It can add/remove proxy clients from this collection,
// and dispatch web socket message to a specific proxy client.
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
func (wsc *WebSocketClient) NewProxy(conn io.ReadWriteCloser, server chan ServerData,
	close chan bool, cherr chan error) *ProxyClient {
	id := ksuid.New()
	proxy := ProxyClient{Id: id, server: server, close: close, cherr: cherr}

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

// remove current proxy by id
func (wsc *WebSocketClient) RemoveProxy(id ksuid.KSUID) {
	wsc.proxyMu.Lock()
	defer wsc.proxyMu.Unlock()
	if _, ok := wsc.proxies[id]; ok {
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
					proxy.close <- false
				case WsTpData:
					var proxyData ProxyData
					if err := json.Unmarshal(socketData, &proxyData); err != nil {
						proxy.cherr <- err
						continue
					}
					if decodeBytes, err := base64.StdEncoding.DecodeString(proxyData.DataBase64); err != nil {
						proxy.cherr <- err
						continue
					} else {
						// just write data back
						proxy.server <- ServerData{Type: proxyData.Type, Data: decodeBytes}
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
