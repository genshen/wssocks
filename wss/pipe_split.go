package wss

import (
	"io"
	"sync"

	"github.com/segmentio/ksuid"
)

type writer = io.Writer

type queue struct {
	masterID ksuid.KSUID            // 主连接ID
	buffer   chan []byte            // 写入缓冲区
	writers  map[ksuid.KSUID]writer // 每一个连接
	sorted   []ksuid.KSUID          // 连接发送的顺序
	status   string                 // 管道状态
	ctime    int64                  // 创建时间
}

func NewQueue(masterID ksuid.KSUID) *queue {
	return &queue{
		masterID: masterID,
		buffer:   makeBuffer(),
		writers:  make(map[ksuid.KSUID]writer),
		status:   "wait",
	}
}

func (q *queue) SetSort(sort []ksuid.KSUID) {
	q.sorted = sort
}

// 写入缓冲区
func (q *queue) Write(data []byte) (n int, err error) {
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			pipePrintln("split.writer recover", err)
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

// 从缓冲区读取并发送到各个连接
func (q *queue) Send() error {
	// 如果已经在发送，返回
	if q.status == "send" {
		return nil
	}
	// 设置为开始发送
	q.status = "send"
	for {
		// 如果状态已经关闭，则返回
		if q.status == "close" {
			return io.EOF
		}
		for _, id := range q.sorted {
			w := q.writers[id]
			data, err := readWithTimeout(q.buffer)
			if err != nil {
				return err
			}
			pipePrintln("split.send to:", id, "data:", string(data))
			_, e := w.Write(data)
			if e != nil {
				pipePrintln("split.send write", e.Error())
				return e
			}
		}
	}
}

// 关闭通道
func (q *queue) Close() {
	q.status = "close"
	close(q.buffer)
}

type QueueHub struct {
	queue map[ksuid.KSUID]*queue
	mu    *sync.RWMutex
}

func NewQueueHub() *QueueHub {
	qh := &QueueHub{
		queue: make(map[ksuid.KSUID]*queue),
		mu:    &sync.RWMutex{},
	}
	return qh
}

// 把连接都加进来，不用保证顺序
func (h *QueueHub) AddWriter(masterID ksuid.KSUID, id ksuid.KSUID, w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 不存在，就先创建
	if _, ok := h.queue[masterID]; !ok {
		h.queue[masterID] = NewQueue(masterID)
	}

	h.queue[masterID].writers[id] = w
}

// 删除写
func (h *QueueHub) DelWriter(masterID ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 存在，就删除
	if q, ok := h.queue[masterID]; ok {
		q.Close()
		delete(h.queue, masterID)
	}
}

// 获取数据
func (h *QueueHub) GetById(masterID ksuid.KSUID) *queue {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.queue[masterID]; ok {
		return q
	}
	return nil
}

// 设置全部连接
func (q *QueueHub) SetSort(masterID ksuid.KSUID, sort []ksuid.KSUID) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q, ok := q.queue[masterID]; ok {
		q.SetSort(sort)
	}
}

// 根据状态决定是否可开启发送
func (h *QueueHub) TrySend(masterID ksuid.KSUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.queue[masterID]; ok {
		if q.status == "send" {
			return true
		}
		if len(q.writers) == len(q.sorted) {
			pipePrintln("split try", q.sorted)
			go q.Send()
			return true
		}
	}
	return false
}
