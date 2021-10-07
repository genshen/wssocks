package wss

import (
	"io"
	"net"
	"sync"

	"github.com/segmentio/ksuid"
)

type queue2 struct {
	buffer chan []byte
	master ksuid.KSUID
	sorted []ksuid.KSUID
	conn   net.Conn
	status string
}

func (q *queue2) SetConn(conn net.Conn) {
	q.conn = conn
}

func (q *queue2) Send(hub *queueHub2) error {
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

func (q *queue2) SetSort(sort []ksuid.KSUID) {
	q.sorted = sort
}

func (q *queue2) setData(data []byte) {
	b := make([]byte, len(data))
	copy(b, data)
	q.buffer <- b
}

func (q *queue2) Close() {
	q.status = "close"
	for _, id := range q.sorted {
		q := outQueueHub.GetById(id)
		close(q.buffer)
	}
}

type queueHub2 struct {
	queue   map[ksuid.KSUID]*queue2
	id2mas  map[ksuid.KSUID]ksuid.KSUID
	counter map[ksuid.KSUID]int64
	status  map[ksuid.KSUID]bool
	mu      *sync.RWMutex
}

func NewQueueHub2() *queueHub2 {
	qh := &queueHub2{
		queue:   make(map[ksuid.KSUID]*queue2),
		id2mas:  make(map[ksuid.KSUID]ksuid.KSUID),
		counter: make(map[ksuid.KSUID]int64),
		status:  make(map[ksuid.KSUID]bool),
		mu:      &sync.RWMutex{},
	}
	return qh
}

func (h *queueHub2) addBufQueue(id ksuid.KSUID, masterId ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.queue[id]; !ok {
		h.queue[id] = &queue2{
			master: masterId,
			buffer: make(chan []byte, 10),
		}
	}
}

func (h *queueHub2) GetById(id ksuid.KSUID) *queue2 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.queue[id]; ok {
		return q
	}
	return nil
}

// 设置与主id的关系，用于查找主id中的conn和sorted
func (h *queueHub2) SetMap(id ksuid.KSUID, master ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.id2mas[id] = master
}

func (h *queueHub2) Incre(id ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.counter[id]; ok {
		h.counter[id]++
	} else {
		h.counter[id] = 1
	}
}

// 服务端根据状态决定发送
func (h *queueHub2) TrySend(id ksuid.KSUID, conn net.Conn) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if s, ok := h.status[id]; ok {
		return s
	}

	if conn != nil {
		h.queue[id].SetConn(conn)
	}

	if c, ok := h.counter[id]; ok {
		if q, ok := h.queue[id]; ok {
			if c == int64(len(q.sorted)) {
				pipePrintln("join try", q.sorted, q.conn)
				if q.conn != nil {
					h.status[id] = true
					go q.Send(h)
					return true
				}
			}
		}
	}
	return false
}
