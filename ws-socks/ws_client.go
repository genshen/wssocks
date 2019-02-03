package ws_socks

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"log"
	"net"
	"sync"
)

type WebSocketClient struct {
	ConcurrentWebSocket
	proxies map[ksuid.KSUID]*Proxy // all proxies on this websocket.
	mu      sync.RWMutex           // mutex to operate proxies map.
}

// get the connection size
func (wsc *WebSocketClient) ConnSize() int {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()
	return len(wsc.proxies)
}

// establish websocket connection
func (wsc *WebSocketClient) Connect(addr string) {
	log.Println("connecting to ", addr)
	ws, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		log.Fatal("establishing connection error:", err)
	}
	wsc.WsConn = ws
	wsc.proxies = make(map[ksuid.KSUID]*Proxy)
}

// create a new proxy with unique id
func (wsc *WebSocketClient) NewProxy(conn *net.TCPConn) *Proxy {
	id := ksuid.New()
	proxy := Proxy{Id: id, Conn: conn}
	proxy.isClosed = false

	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	wsc.proxies[id] = &proxy
	return &proxy
}

func (wsc *WebSocketClient) GetProxyById(id ksuid.KSUID) *Proxy {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()
	if proxy, ok := wsc.proxies[id]; ok {
		return proxy
	}
	return nil
}

// listen income websocket message and dispatch to different proxies.
func (wsc *WebSocketClient) ListenIncomeMsg() {
	for {
		_, data, err := wsc.WsConn.ReadMessage()
		if err != nil {
			log.Println("error websocket read:", err) // todo close all
			return
		}

		var socketData json.RawMessage
		socketStream := WebSocketMessage{
			Data: &socketData,
		}
		if err := json.Unmarshal(data, &socketStream); err != nil {
			continue // todo log
		} else {
			// find proxy by id
			id := socketStream.Id
			if ksid, err := ksuid.Parse(id); err != nil {
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
							continue
						}
						if err := proxy.DispatchData(&proxyData); err != nil {
							wsc.Close(ksid)
							continue
						}
					}
				}
			}
		}
	}
}

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

func (wsc *WebSocketClient) Close(id ksuid.KSUID) {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	if proxy, ok := wsc.proxies[id]; ok {
		proxy.Close()
		delete(wsc.proxies, id)
	}
}
