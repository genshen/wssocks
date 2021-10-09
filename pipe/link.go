package pipe

// 将从多个连接收到的数据先存在各自的buffer, 再排序发送到对外的连接conn

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/segmentio/ksuid"
)

type link struct {
	master  ksuid.KSUID // 主连接ID，用于反向查找主连接数据
	buffer  chan buffer // 连接数据缓冲区
	status  string      // 当前状态
	done    chan struct{}
	sorted  []ksuid.KSUID // 当前为主连接时存储所有连接标识
	conn    *net.TCPConn  // 当前为主连接时存储目标连接
	counter int           // 当前为主连接时存储已经收到的连接计数，用于决定是否可以向外发送数据
	ctime   time.Time     // 创建时间
}

func NewLink(masterID ksuid.KSUID) *link {
	return &link{
		master:  masterID,
		buffer:  makeBuffer(),
		status:  StaWait,
		done:    make(chan struct{}),
		counter: 0,
		ctime:   time.Now(),
	}
}

// 设置对外连接，仅当前为主连接调用
func (q *link) SetConn(conn *net.TCPConn) {
	q.conn = conn
}

// 设置排序，仅当前为主连接调用
func (q *link) SetSort(sort []ksuid.KSUID) {
	q.sorted = sort
}

// 设置EOF
func (q *link) WriteEOF() {
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			pipePrintln("link.writer recover", err)
			return
		}
	}()
	q.buffer <- buffer{eof: true, data: []byte{}}
}

// 发送数据
func (q *link) Send(hub *LinkHub) error {
	// 如果已经在发送，返回
	if q.status == StaSend {
		return nil
	}
	defer func() {
		q.done <- struct{}{}
		close(q.done)
	}()
	// 设置为开始发送
	q.status = StaSend
	for {
		// 用于循环中的退出
		if q.status == StaClose {
			return io.ErrClosedPipe
		}
		for _, id := range q.sorted {
			if q, ok := hub.links[id]; ok {
				var conn *net.TCPConn
				if id == q.master {
					conn = q.conn
				} else if mq, ok := hub.links[q.master]; ok {
					conn = mq.conn
				}

				b, err := readWithTimeout(q.buffer, expTenMinute)
				if err != nil {
					return err
				}
				if b.eof {
					// 发送关闭写请求
					conn.CloseWrite()
					return nil
				}
				if conn != nil {
					pipePrintln("join.send from:", id, "send:", string(b.data))
					_, e := conn.Write(b.data)
					if e != nil {
						pipePrintln("join.send write", e.Error())
						return e
					}
				} else {
					pipePrintln(id, "join.send conn not found")
					return errors.New("conn not found")
				}
			} else {
				pipePrintln(id, "join.send queue not found")
				return errors.New("queue not found")
			}
		}
	}
}

// 写入缓冲区数据
func (q *link) Write(data []byte) (n int, err error) {
	defer func() {
		// 捕获异常
		if err := recover(); err != nil {
			pipePrintln("join.write recover", err)
			return
		}
	}()
	b := make([]byte, len(data))
	copy(b, data)
	q.buffer <- buffer{eof: false, data: b}
	return len(data), nil
}

// 堵塞等待
func (q *link) Wait() {
	<-q.done
}

// 释放资源
func (q *link) close(id ksuid.KSUID) {
	if q.status == StaClose {
		return
	}
	q.status = StaClose
	if q.buffer != nil {
		close(q.buffer)
	}
	// 是主连接
	if id == q.master && q.conn != nil {
		q.conn.Close()
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

// 增加连接
func (h *LinkHub) Add(id ksuid.KSUID, masterID ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 先初始化主连接做计数器用
	m, ok := h.links[masterID]
	if !ok {
		h.links[masterID] = NewLink(masterID)
		m = h.links[masterID]
	}

	// 所有连接
	if _, ok := h.links[id]; !ok {
		h.links[id] = NewLink(masterID)
	}
	m.counter++
	h.trySend(masterID, nil)
}

// 删除连接
func (h *LinkHub) RemoveAll(masterID ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 存在，就删除
	if q, ok := h.links[masterID]; ok {
		for _, id := range q.sorted {
			if id == masterID {
				continue
			}
			if c, ok := h.links[id]; ok {
				c.close(id)
				delete(h.links, id)
			}
		}
		q.close(masterID)
		delete(h.links, masterID)
	}
}

// 取数据
func (h *LinkHub) Get(id ksuid.KSUID) *link {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.links[id]; ok {
		return q
	}
	return nil
}

// 写数据
func (h *LinkHub) Write(id ksuid.KSUID, data []byte) (n int, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if q, ok := h.links[id]; ok {
		return q.Write(data)
	}
	return 0, errors.New("join.hub link not found")
}

// 设置连接传输顺序
func (h *LinkHub) SetSort(masterID ksuid.KSUID, sort []ksuid.KSUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if q, ok := h.links[masterID]; ok {
		q.SetSort(sort)
	} else {
		h.links[masterID] = NewLink(masterID)
		h.links[masterID].SetSort(sort)
	}
}

// 服务端根据状态决定发送
func (h *LinkHub) trySend(masterID ksuid.KSUID, conn *net.TCPConn) bool {
	if q, ok := h.links[masterID]; ok {
		if conn != nil {
			q.SetConn(conn)
		}
		//fmt.Println("try", q.conn, q.counter, len(q.sorted))
		if q.conn != nil && q.counter == len(q.sorted) {
			pipePrintln("join.hub try", q.sorted, q.conn)
			go q.Send(h)
			return true
		}
	}
	return false
}

// 服务端根据状态决定发送
func (h *LinkHub) TrySend(masterID ksuid.KSUID, conn *net.TCPConn) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.trySend(masterID, conn)
}

// 删除过期数据
func (h *LinkHub) TimeoutClose() {
	h.mu.Lock()
	defer h.mu.Unlock()

	var tmp []ksuid.KSUID
	for id, link := range h.links {
		if time.Since(link.ctime) > expHour {
			pipePrintln("join.hub timeout", id, link.ctime.String())
			tmp = append(tmp, id)
			if len(tmp) > 100 { //单次最大处理条数
				break
			}
		}
	}
	for _, id := range tmp {
		h.links[id].close(id)
		delete(h.links, id)
	}
}
