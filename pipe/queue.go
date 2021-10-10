package pipe

// 将收到的数据先放到buffer，再异步写入多个向外的连接(writer)

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/segmentio/ksuid"
)

type queue struct {
	masterID ksuid.KSUID                // 主连接ID
	buffer   chan buffer                // 写入缓冲区
	writers  map[ksuid.KSUID]PipeWriter // 每一个连接
	sorted   []ksuid.KSUID              // 连接发送的顺序
	status   string                     // 管道状态
	done     chan struct{}
	ctime    time.Time // 创建时间
}

func NewQueue(masterID ksuid.KSUID) *queue {
	return &queue{
		masterID: masterID,
		buffer:   makeBuffer(),
		writers:  make(map[ksuid.KSUID]PipeWriter),
		status:   StaWait,
		done:     make(chan struct{}),
		ctime:    time.Now(),
	}
}

// 设置顺序
func (q *queue) SetSort(sort []ksuid.KSUID) {
	q.sorted = sort
}

// 写入缓冲区
func (q *queue) Write(data []byte) (n int, err error) {
	b := make([]byte, len(data))
	copy(b, data)
	return q.writeBuf(buffer{eof: false, data: b})
}

// 发送EOF
func (q *queue) WriteEOF() {
	q.writeBuf(buffer{eof: true, data: []byte{}})
}

// 基础方法
func (q *queue) writeBuf(b buffer) (int, error) {
	if q.status == StaDone {
		return 0, errors.New("send is over")
	}

	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			pipePrintln("queue.writer recover", err)
			return
		}
	}()
	select {
	case <-time.After(expFiveMinute):
		return 0, errors.New("write timeout")
	case q.buffer <- b:
	}
	return len(b.data), nil
}

// 从缓冲区读取并发送到各个连接
func (q *queue) Send() error {
	// 如果已经在发送，返回
	if q.status == StaSend || q.status == StaDone {
		return nil
	}
	defer func() {
		q.status = StaDone
		q.done <- struct{}{}
		close(q.done)
	}()
	// 设置为开始发送
	q.status = StaSend
	for {
		// 如果状态已经关闭，则返回
		if q.status == StaClose {
			return io.ErrClosedPipe
		}
		for _, id := range q.sorted {
			w := q.writers[id]
			b, err := readWithTimeout(q.buffer, expFiveMinute)
			if err != nil {
				return err
			}
			if b.eof {
				//fmt.Println("queue send eof")
				w.WriteEOF()
				return nil
			}
			pipePrintln("queue.send to:", id, "data:", string(b.data))
			_, e := w.Write(b.data)
			if e != nil {
				pipePrintln("queue.send write", e.Error())
				return e
			}
		}
	}
}

// 堵塞等待
func (q *queue) Wait() {
	<-q.done
}

// 关闭通道
func (q *queue) close() {
	if q.status == StaClose {
		return
	}
	q.status = StaClose
	if q.buffer != nil {
		close(q.buffer)
	}
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
func (h *QueueHub) Add(masterID ksuid.KSUID, id ksuid.KSUID, w PipeWriter) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 不存在，就先创建
	if _, ok := h.queue[masterID]; !ok {
		h.queue[masterID] = NewQueue(masterID)
	}

	h.queue[masterID].writers[id] = w
	h.trySend(masterID)
}

// 删除写
func (h *QueueHub) Remove(masterID ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 存在，就删除
	if q, ok := h.queue[masterID]; ok {
		q.close()
		delete(h.queue, masterID)
	}
}

// 获取数据
func (h *QueueHub) Get(masterID ksuid.KSUID) *queue {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.queue[masterID]; ok {
		return q
	}
	return nil
}

// 设置全部连接
func (h *QueueHub) SetSort(masterID ksuid.KSUID, sort []ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if q, ok := h.queue[masterID]; ok {
		q.SetSort(sort)
	} else {
		h.queue[masterID] = NewQueue(masterID)
		h.queue[masterID].SetSort(sort)
	}
}

// 根据状态决定是否可开启发送
func (h *QueueHub) trySend(masterID ksuid.KSUID) bool {
	if q, ok := h.queue[masterID]; ok {
		if len(q.writers) == len(q.sorted) {
			pipePrintln("queue try", q.sorted)
			go q.Send()
			return true
		}
	}
	return false
}

// 根据状态决定是否可开启发送
func (h *QueueHub) TrySend(masterID ksuid.KSUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.trySend(masterID)
}

// 删除过期数据
func (h *QueueHub) TimeoutClose() {
	h.mu.Lock()
	defer h.mu.Unlock()

	var tmp []ksuid.KSUID
	for id, queue := range h.queue {
		if time.Since(queue.ctime) > expHour {
			pipePrintln("queue.hub timeout", id, queue.ctime.String())
			tmp = append(tmp, id)
			if len(tmp) > 100 { //单次最大处理条数
				break
			}
		}
	}
	for _, id := range tmp {
		h.queue[id].close()
		delete(h.queue, id)
	}
}

// 获取数据
func (h *QueueHub) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.queue)
}
