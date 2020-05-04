package wss

import (
	"bytes"
    "errors"
	"io"
	"sync"
)

type BufferedWR struct {
	buffer bytes.Buffer
	done   bool
	//closeCh bool
	update chan struct{}
	mu     sync.Mutex
}

func (h *BufferedWR) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return nil
	}
	h.done = true
    close(h.update)
	return nil
}

func (h *BufferedWR) isClosed() bool {
	return h.done
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (h *BufferedWR) Write(p []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
    if h.done {
        return 0, errors.New("write after buffer closed")
    }
    // make sure it indeed has data in buffer when noticing wait
    if len(p) == 0 {
        return 0, nil
    }
	if h.buffer.Len() == 0 {
		h.update <- struct{}{}
	}
	return h.buffer.Write(p)
}

// read data from buffer
// make sure there is no more one goroutine reading
func (h *BufferedWR) Read(p []byte) (int, error) {
    // wait to make sure there is data in buffer
    if h.buffer.Len() == 0 {
        select {
        case _, ok := <-h.update: // data received from client
            if !ok {
                return 0, io.EOF
            }
        }
    }

    h.mu.Lock()
    defer h.mu.Unlock()
    if h.done {
        return 0, io.EOF
	}
    return h.buffer.Read(p)
}

func NewBufferWR() *BufferedWR {
	update := make(chan struct{}, 1)
    body := BufferedWR{done: false, update: update}
	return &body
}
