package wss

import (
    "context"
    "github.com/segmentio/ksuid"
    "nhooyr.io/websocket"
    "nhooyr.io/websocket/wsjson"
    "sync"
)

type ProxyServer struct {
    Id       ksuid.KSUID // id of proxy connection
    ProxyIns ProxyEstablish
}

// Hub maintains the set of active proxy clients in server side for a user
type Hub struct {
    ConcurrentWebSocket
    // Registered proxy connections.
    connPool map[ksuid.KSUID]*ProxyServer

    // establish connection based on the request from client side.
    est chan ProxyRegister

    // register proxy connection
    register chan *ProxyServer

    // Unregister requests from clients.
    unregister chan ksuid.KSUID

    tellClose chan ksuid.KSUID

    mu sync.RWMutex
}

type ProxyRegister struct {
    id       ksuid.KSUID
    _type    int
    addr     string
    withData []byte
}

func NewHub(ctx context.Context, conn *websocket.Conn) *Hub {
    return &Hub{
        ConcurrentWebSocket: ConcurrentWebSocket{WsConn: conn},
        est:                 make(chan ProxyRegister),
        register:            make(chan *ProxyServer),
        unregister:          make(chan ksuid.KSUID),
        connPool:            make(map[ksuid.KSUID]*ProxyServer),
        tellClose:           make(chan ksuid.KSUID),
    }
}

func (h *Hub) Close() {
    // if there are connections, close them.
    h.mu.Lock()
    defer h.mu.Unlock()
    for id, proxy := range h.connPool {
        proxy.ProxyIns.Close(false)
        delete(h.connPool, id)
    }
    close(h.est)
    close(h.register)
    close(h.unregister)
    close(h.tellClose)
}

func (h *Hub) Run() {
    for {
        select {
        case estProxy, ok := <-h.est:
            if !ok {
                break
            }
            go establishProxy(h, estProxy)

        case proxy, ok := <-h.register:
            if !ok {
                break
            }
            h.addNewProxy(proxy)
        case id, ok := <-h.unregister:
            if !ok {
                break
            }
            if proxy := h.GetProxyById(id); proxy != nil {
                proxy.ProxyIns.Close(false) // todo remove proxy here
            }
        case id := <-h.tellClose: // send close message to proxy client
            h.tellClosed(id)
        }
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

func (h *Hub) GetConnectorSize() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.connPool)
}

// remove a connection specified by id.
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
