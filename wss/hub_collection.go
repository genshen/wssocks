package wss

import (
    "github.com/segmentio/ksuid"
    "nhooyr.io/websocket"
    "sync"
)

// HubCollection is a set of hubs. It handle several hubs.
// Each hub can map to a websocket connection,
// which also handle several proxies instance.
type HubCollection struct {
    hubs map[ksuid.KSUID]*Hub

    mutex sync.RWMutex
}

func NewHubCollection() *HubCollection {
    hc := HubCollection{}
    hc.hubs = make(map[ksuid.KSUID]*Hub)
    return &hc
}

// add a hub to hub collection
func (hc *HubCollection) AddHub(conn *websocket.Conn) *Hub {
    hc.mutex.Lock()
    defer hc.mutex.Unlock()

    hub := Hub{
        id:                  ksuid.New(),
        ConcurrentWebSocket: ConcurrentWebSocket{WsConn: conn},
        est:                 make(chan ProxyRegister),
        register:            make(chan *ProxyServer),
        unregister:          make(chan ksuid.KSUID),
        connPool:            make(map[ksuid.KSUID]*ProxyServer),
    }

    hc.hubs[hub.id] = &hub
    return &hub
}

// remove a hub specified by its id.
func (hc *HubCollection) RemoveProxy(id ksuid.KSUID) {
    hc.mutex.Lock()
    defer hc.mutex.Unlock()
    if _, ok := hc.hubs[id]; ok {
        delete(hc.hubs, id)
    }
}
