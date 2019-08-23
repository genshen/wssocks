package wss

import (
	"encoding/base64"
	"github.com/genshen/wssocks/wss/term_view"
	"github.com/genshen/wssocks/wss/ticker"
	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
)

// proxy client handle one connection, send data to proxy server vai websocket.
type ProxyClient struct {
	Conn     *net.TCPConn
	Id       ksuid.KSUID
	isClosed bool
}

// can't do large compute or communication here
func (p *ProxyClient) DispatchData(data *ProxyData) error {
	// decode base64
	if decodeBytes, err := base64.StdEncoding.DecodeString(data.DataBase64); err != nil { // todo ignore error
		log.Error("base64 decode error,", err)
		return err // skip error
	} else {
		// just write data back
		if _, err := p.Conn.Write(decodeBytes); err != nil {
			return err
		}
	}
	return nil
}

// close (tcp) connection
// the close command can be from server
func (p *ProxyClient) Close() {
	if p.isClosed {
		return
	}
	p.Conn.Close()
	p.isClosed = true
}

// handel socket dial results processing
// copy income connection data to proxy serve via websocket
func (p *ProxyClient) Serve(plog *term_view.ProgressLog, wsc *WebSocketClient,
	tick *ticker.Ticker, proxyType int, addr string) error {
	plog.Update(term_view.Status{IsNew: true, Address: addr})
	defer plog.Update(term_view.Status{IsNew: false, Address: addr})
	defer wsc.Close(p.Id)

	addrSend := WebSocketMessage{Type: WsTpEst, Id: p.Id.String(),
		Data: ProxyEstMessage{Type: proxyType, Addr: addr}}
	if err := wsc.WriteWSJSON(&addrSend); err != nil {
		log.Error("json error:", err)
		return err
	}

	if tick != nil {
		var buffer Base64WSBufferWriter
		defer buffer.Flush(websocket.TextMessage, p.Id, &(wsc.ConcurrentWebSocket))

		defer tick.Remove(ticker.TickId(p.Id))
		tick.Append(ticker.TickId(p.Id), func() { // fixme return error
			_, err := buffer.Flush(websocket.TextMessage, p.Id, &(wsc.ConcurrentWebSocket))
			if err != nil {
				log.Error("write error:", err) // todo use of closed network connection
			}
		}) // todo p.id

		if _, err := io.Copy(&buffer, p.Conn); err != nil { // copy data to buffer
			log.Error("io copy error,", err)
			return nil
		}
	} else {
		// dont use ticker
		var buffer = make([]byte, 1024*64)
		for {
			if n, err := p.Conn.Read(buffer); err != nil {
				break
				// log.Println("read error:", err)
			} else if n > 0 {
				if err := wsc.WriteProxyMessage(p.Id, buffer[:n]); err != nil {
					log.Error("write error:", err)
					break
				}
			}
		}
	}
	return nil
}
