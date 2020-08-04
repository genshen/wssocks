package wss

import (
    "context"
	"encoding/base64"
	"encoding/json"
	"github.com/segmentio/ksuid"
	"net/http"
    "nhooyr.io/websocket"
    "nhooyr.io/websocket/wsjson"
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
    cancel context.CancelFunc
}

// get the connection size
func (wsc *WebSocketClient) ConnSize() int {
	wsc.proxyMu.RLock()
	defer wsc.proxyMu.RUnlock()
	return len(wsc.proxies)
}

// Establish websocket connection.
// And initialize proxies container.
func NewWebSocketClient(ctx context.Context, addr string, hc *http.Client, header http.Header) (*WebSocketClient, error) {
    ws, _, err := websocket.Dial(ctx, addr, &websocket.DialOptions{HTTPClient: hc, HTTPHeader: header})
	if err != nil {
		return nil, err
	}
    return &WebSocketClient{
        ConcurrentWebSocket: ConcurrentWebSocket{
            WsConn: ws,
        },
        cancel:  nil,
        proxies: make(map[ksuid.KSUID]*ProxyClient),
	}, nil
}

// create a new proxy with unique id
func (wsc *WebSocketClient) NewProxy(onData func(ksuid.KSUID, ServerData),
	onClosed func(ksuid.KSUID, bool), onError func(ksuid.KSUID, error)) *ProxyClient {
	id := ksuid.New()
	proxy := ProxyClient{Id: id, onData: onData, onClosed: onClosed, onError: onError}

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
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := wsjson.Write(ctx, wsc.WsConn, &finish); err != nil {
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
func (wsc *WebSocketClient) ListenIncomeMsg(readLimit int64) error {
    ctx, can := context.WithCancel(context.Background())
    wsc.cancel = can
	wsc.WsConn.SetReadLimit(readLimit)

	for {
		// check stop first
		select {
        case <-ctx.Done():
			return StoppedError
		default:
			// if the channel is still open, continue as normal
		}

        _, data, err := wsc.WsConn.Read(ctx)
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
					proxy.onClosed(ksid, false)
				case WsTpData:
					var proxyData ProxyData
					if err := json.Unmarshal(socketData, &proxyData); err != nil {
						proxy.onError(ksid, err)
						continue
					}
					if decodeBytes, err := base64.StdEncoding.DecodeString(proxyData.DataBase64); err != nil {
						proxy.onError(ksid, err)
						continue
					} else {
						// just write data back
						proxy.onData(ksid, ServerData{Tag: proxyData.Tag, Data: decodeBytes})
					}
				}
			}
		}
	}
}

func (wsc *WebSocketClient) Close() error {
    if wsc.cancel != nil {
        wsc.cancel()
    }
	if err := wsc.WSClose(); err != nil {
		return err
	}
	return nil
}
