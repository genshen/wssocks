package wss

import (
	"io"
	"sync"

	"github.com/segmentio/ksuid"
)

type writer = io.Writer

type queue struct {
	masterID ksuid.KSUID            // 主连接
	buffer   chan []byte            //
	writers  map[ksuid.KSUID]writer //每一个连接
	sorted   []ksuid.KSUID          //连接请求的顺序
	counter  int64                  //计数器
	status   string
}

func (q *queue) Write(data []byte) (n int, err error) {
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			pipePrintln("split recover", err)
			return
		}
	}()
	//pipePrintln("write to chan", q.id, string(buffer))
	b := make([]byte, len(data))
	copy(b, data)
	q.buffer <- b
	//pipePrintln("write ok", string(buffer))
	return len(data), nil
}

func (q *queue) Send() error {
	q.status = "send"
	for {
		if q.status == "close" {
			return io.EOF
		}
		for _, id := range q.sorted {
			w := q.writers[id]
			data, err := readWithTimeout(q.buffer)
			if err != nil {
				return err
			}
			pipePrintln("split send to:", id, "data:", string(data))
			_, e := w.Write(data)
			if e != nil {
				pipePrintln("writer.Write", e.Error())
				return e
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
	mu      *sync.RWMutex
}

func NewQueueHub() *queueHub {
	qh := &queueHub{
		queue: make(map[ksuid.KSUID]*queue),
		mu:    &sync.RWMutex{},
	}
	return qh
}

// 不保证顺序
func (h *queueHub) addWriter(masterID ksuid.KSUID, curID ksuid.KSUID, w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.queue[masterID]; !ok {
		h.queue[masterID] = &queue{
			masterID: masterID,
			buffer:   make(chan []byte, 10),
			writers:  make(map[ksuid.KSUID]writer),
			counter:  1,
			status:   "wait",
		}
	}

	h.queue[masterID].writers[curID] = w
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
	if q, ok := h.queue[id]; ok {
		q.counter++
	} else {
		q.counter = 1
	}
}

// 服务端根据状态决定发送
func (h *queueHub) TrySend(id ksuid.KSUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.queue[id]; ok {
		if q.status == "send" {
			return true
		}
		if q.counter == int64(len(q.sorted)) {
			pipePrintln("split try", q.sorted)
			go q.Send()
			return true
		}
	}
	return false
}
