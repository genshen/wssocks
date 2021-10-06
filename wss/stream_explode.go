package wss

import (
	"fmt"
	"io"
	"sync"

	"github.com/segmentio/ksuid"
)

type tunnel struct {
	//tcpConn net.Conn
	writer io.Writer
}

type queue struct {
	id     ksuid.KSUID
	buffer chan []byte
	cur    int
	tunnel map[ksuid.KSUID]tunnel
	sorted []ksuid.KSUID
	status string
}

func (q *queue) Write(buffer []byte) (n int, err error) {
	fmt.Println("write to chan", q.id, string(buffer))
	b := make([]byte, len(buffer))
	copy(b, buffer)
	q.buffer <- b
	fmt.Println("write ok", string(buffer))
	return len(buffer), nil
}

func (q *queue) Send() {
	for {
		if q.status == "close" {
			return
		}
		for _, id := range q.sorted {
			t := q.tunnel[id]
			data := <-q.buffer
			fmt.Println("tunnel send:", id, string(data), q.id)
			_, e := t.writer.Write(data)
			if e != nil {
				fmt.Println("writer.Write", e.Error())
				return
			}
		}
	}
}

func (q *queue) SetSort(sort []ksuid.KSUID) {
	q.sorted = sort
}
func (q *queue) Close() {
	q.status = "close"
	close(q.buffer)
}

type queueHub struct {
	queue   map[ksuid.KSUID]*queue
	counter map[ksuid.KSUID]int64
	status  map[ksuid.KSUID]bool
	mu      *sync.RWMutex
}

func NewQueueHub() *queueHub {

	qh := &queueHub{
		queue:   make(map[ksuid.KSUID]*queue),
		counter: make(map[ksuid.KSUID]int64),
		status:  make(map[ksuid.KSUID]bool),
		mu:      &sync.RWMutex{},
	}
	return qh
}

// 不保证顺序
func (h *queueHub) addWriter(id ksuid.KSUID, id2 ksuid.KSUID, writer io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.queue[id]; !ok {
		h.queue[id] = &queue{
			cur:    0,
			id:     id,
			buffer: make(chan []byte, 1),
			tunnel: make(map[ksuid.KSUID]tunnel),
		}
	}

	t := tunnel{writer: writer}
	h.queue[id].tunnel[id2] = t
}

func (h *queueHub) GetById(id ksuid.KSUID) *queue {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.queue[id]; ok {
		return q
	}
	return nil
}

func (h *queueHub) Incre(id ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.counter[id]; ok {
		h.counter[id]++
	} else {
		h.counter[id] = 1
	}
}

// 服务端根据状态决定发送
func (h *queueHub) TrySend(id ksuid.KSUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if s, ok := h.status[id]; ok {
		return s
	}

	if c, ok := h.counter[id]; ok {
		if q, ok := h.queue[id]; ok {
			if c == int64(len(q.sorted)) {
				fmt.Println("tunnel try", c, q.sorted)
				h.status[id] = true
				go q.Send()
				return true
			}
		}
	}
	return false
}
