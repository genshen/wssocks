package ws_socks

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"log"
	"net"
)

type WebSocketClient struct {
	ConcurrentWebSocket
	proxies map[ksuid.KSUID]*Proxy
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
func (wsc *WebSocketClient) NewProxy(conn *net.TCPConn, ) *Proxy {
	id := ksuid.New()
	proxy := Proxy{Id: id, Conn: conn}
	wsc.proxies[id] = &proxy
	return &proxy
}

// listen income websocket message and dispatch to different proxies.
func (wsc *WebSocketClient) ListenIncomeMsg() {
	for {
		_, data, err := wsc.WsConn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		var socketData json.RawMessage
		socketStream := WebSocketMessage2{
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
				if proxy, ok := wsc.proxies[ksid]; !ok {
					continue
				} else {
					// now, we known the id and type of incoming data
					switch socketStream.Type {
					case WsTpClose: // remove proxy
						proxy.Close()
						delete(wsc.proxies, ksid)
						// todo notice closing
					case WsTpData:
						var proxyData ProxyData
						if err := json.Unmarshal(socketData, &proxyData); err != nil {
							continue
						}
						proxy.DispatchData(&proxyData) // todo error, e.g connection closed
					}
				}

			}
		}
	}
}
