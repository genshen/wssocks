package wss

import (
	"bytes"
	"io"
	"sync"
)

type BufferedWR struct {
	buffer bytes.Buffer
	done   bool
	//closeCh bool
	update chan struct{}
	close  chan struct{}
	mu     sync.Mutex
}

func (h *BufferedWR) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.done {
		return nil
	}
	h.done = true
	h.close <- struct{}{}
	return nil
}

func (h *BufferedWR) isClosed() bool {
	return h.done
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (h *BufferedWR) Write(p []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.buffer.Len() == 0 {
		h.update <- struct{}{}
	}
	return h.buffer.Write(p)
}

func (h *BufferedWR) Read(p []byte) (int, error) {
RERUN:
	h.mu.Lock()
	if h.done {
		h.mu.Unlock()
		return 0, io.EOF
	}

	if h.buffer.Len() != 0 {
		for i, b := range h.buffer.Bytes() {
			p[i] = b
		}
		l := h.buffer.Len()
		h.buffer.Reset()
		h.mu.Unlock()
		return l, nil
	}
	h.mu.Unlock()
	// if buffer is empty
	select {
	case <-h.close: // close from client
		goto RERUN
	case <-h.update: // data received from client
		goto RERUN
	}
}

func NewBufferWR() *BufferedWR {
	update := make(chan struct{}, 1)
	clo := make(chan struct{}, 1)
	body := BufferedWR{done: false, close: clo, update: update}
	return &body
}
