package wss

import (
	"github.com/segmentio/ksuid"
	"time"
)

type HeartBeat struct {
	wsc      *WebSocketClient
	done     chan bool
	isClosed bool
}

func NewHeartBeat(wsc *WebSocketClient) *HeartBeat {
	hb := HeartBeat{wsc: wsc, isClosed: false}
	hb.done = make(chan bool)
	return &hb
}

// close heartbeat sending
func (hb *HeartBeat) Close() {
	if hb.isClosed {
		return
	}
	hb.done <- true
	close(hb.done)
}

// start sending heart beat to server.
func (hb *HeartBeat) Start() error {
	t := time.NewTicker(time.Second * 15)
	defer t.Stop()
	for {
		select {
		case <-hb.done:
			return nil
		case <-t.C:
			heartBeats := WebSocketMessage{
				Id:   ksuid.KSUID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}.String(),
				Type: WsTpBeats,
				Data: nil,
			}
			if err := hb.wsc.WriteWSJSON(heartBeats); err != nil {
				return err
			}
		}
	}
	return nil
}
