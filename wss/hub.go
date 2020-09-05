package wss

import (
	"context"
	"github.com/segmentio/ksuid"
	"nhooyr.io/websocket/wsjson"
	"sync"
)

type ProxyServer struct {
	Id       ksuid.KSUID // id of proxy connection
	ProxyIns ProxyEstablish
}

// Hub maintains the set of active proxy clients in server side for a user
type Hub struct {
	id ksuid.KSUID
	ConcurrentWebSocket
	// Registered proxy connections.
	connPool map[ksuid.KSUID]*ProxyServer

	mu sync.RWMutex
}

type ProxyRegister struct {
	id       ksuid.KSUID
	_type    int
	addr     string
	withData []byte
}

func (h *Hub) Close() {
	// if there are connections, close them.
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, proxy := range h.connPool {
		proxy.ProxyIns.Close(false)
		delete(h.connPool, id)
	}
}

// add a tcp connection to connection pool.
func (h *Hub) addNewProxy(proxy *ProxyServer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connPool[proxy.Id] = proxy
}

func (h *Hub) GetProxyById(id ksuid.KSUID) *ProxyServer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if proxy, ok := h.connPool[id]; ok {
		return proxy
	}
	return nil
}

// return the proxies handled by this hub/websocket connetion
func (h *Hub) GetConnectorSize() int {
	// h.mu.RLock()
	// defer h.mu.RUnlock()
	return len(h.connPool)
}

// Close proxy connection with remote host.
// It can be called when receiving tell close message from client
func (h *Hub) CloseProxyConn(id ksuid.KSUID) error {
	if proxy := h.GetProxyById(id); proxy != nil {
		return proxy.ProxyIns.Close(false) // todo remove proxy here
	}
	return nil
}

func (h *Hub) RemoveProxy(id ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.connPool[id]; ok {
		delete(h.connPool, id)
	}
}

// tell the client the connection has been closed
func (h *Hub) tellClosed(id ksuid.KSUID) error {
	// send finish flag to client
	finish := WebSocketMessage{
		Id:   id.String(),
		Type: WsTpClose,
		Data: nil,
	}
	// fixme lock or NextWriter
	if err := wsjson.Write(context.TODO(), h.WsConn, &finish); err != nil {
		return err
	}
	return nil
}
