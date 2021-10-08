package wss

import (
	"io"
	"net"
	"sync"

	"github.com/segmentio/ksuid"
)

type link struct {
	buffer  chan []byte
	master  ksuid.KSUID
	sorted  []ksuid.KSUID
	conn    net.Conn
	status  string
	counter int
	ctime   int64 // 创建时间
}

func NewLink(masterID ksuid.KSUID) *link {
	return &link{
		master:  masterID,
		buffer:  makeBuffer(),
		counter: 1,
		status:  "wait",
	}
}

func (q *link) SetConn(conn net.Conn) {
	q.conn = conn
}

func (q *link) SetSort(sort []ksuid.KSUID) {
	q.sorted = sort
}

func (q *link) Send(hub *LinkHub) error {
	// 如果已经在发送，返回
	if q.status == "send" {
		return nil
	}
	// 设置为开始发送
	q.status = "send"
	for {
		if q.status == "close" {
			return io.EOF
		}
		for _, id := range q.sorted {
			q := hub.GetById(id)
			if q == nil {
				pipePrintln(id, "join queue not found")
				continue
			}
			data := <-q.buffer
			mq := hub.GetById(q.master)
			pipePrintln("join from:", id, "send:", string(data))
			_, e := mq.conn.Write(data)
			if e != nil {
				pipePrintln("writer.Write", e.Error())
				return e
			}
		}
	}
}

func (q *link) setData(data []byte) {
	b := make([]byte, len(data))
	copy(b, data)
	q.buffer <- b
}

func (q *link) Close() {
	q.status = "close"
	for _, id := range q.sorted {
		q := outQueueHub.GetById(id)
		close(q.buffer)
	}
}

type LinkHub struct {
	links map[ksuid.KSUID]*link
	mu    *sync.RWMutex
}

func NewLinkHub() *LinkHub {
	qh := &LinkHub{
		links: make(map[ksuid.KSUID]*link),
		mu:    &sync.RWMutex{},
	}
	return qh
}

func (h *LinkHub) addLink(id ksuid.KSUID, masterID ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 所有连接
	if _, ok := h.links[id]; !ok {
		h.links[id] = NewLink(masterID)
	}
	// 防止计数器多算
	if id == masterID {
		return
	}
	// 主连接做计数器加加
	m, ok := h.links[masterID]
	if !ok {
		h.links[masterID] = NewLink(masterID)
	} else {
		m.counter++
	}
}

// 取数据
func (h *LinkHub) GetById(id ksuid.KSUID) *link {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.links[id]; ok {
		return q
	}
	return nil
}

// 设置连接传输顺序
func (h *LinkHub) SetSort(masterID ksuid.KSUID, sort []ksuid.KSUID) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.links[masterID]; ok {
		q.SetSort(sort)
	}
}

// 服务端根据状态决定发送
func (h *LinkHub) TrySend(masterID ksuid.KSUID, conn net.Conn) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if q, ok := h.links[masterID]; ok {
		if conn != nil {
			q.SetConn(conn)
		}
		if q.conn != nil && q.counter == len(q.sorted) {
			pipePrintln("join try", q.sorted, q.conn)
			go q.Send(h)
			return true
		}
	}
	return false
}
