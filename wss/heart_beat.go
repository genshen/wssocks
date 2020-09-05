package wss

import (
	"context"
	"github.com/segmentio/ksuid"
	"nhooyr.io/websocket/wsjson"
	"time"
)

type HeartBeat struct {
	wsc      *WebSocketClient
	cancel   context.CancelFunc
	isClosed bool
}

func NewHeartBeat(wsc *WebSocketClient) (*HeartBeat, context.Context) {
	hb := HeartBeat{wsc: wsc, isClosed: false}
	ctx, can := context.WithCancel(context.Background())

	hb.cancel = can
	return &hb, ctx
}

// close heartbeat sending
func (hb *HeartBeat) Close() {
	if hb.isClosed {
		return
	}
	hb.isClosed = true
	hb.cancel()
}

// start sending heart beat to server.
func (hb *HeartBeat) Start(ctx context.Context, writeTimeout time.Duration) error {
	t := time.NewTicker(time.Second * 15)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			heartBeats := WebSocketMessage{
				Id:   ksuid.KSUID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}.String(),
				Type: WsTpBeats,
				Data: nil,
			}
			writeCtx, _ := context.WithTimeout(ctx, writeTimeout)
			if err := wsjson.Write(writeCtx, hb.wsc.WsConn, heartBeats); err != nil {
				return err
			}
		}
	}
	return nil
}
