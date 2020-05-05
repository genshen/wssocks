package wss

import (
	"github.com/sirupsen/logrus"
	"io"
	"sync"
)

// struct record the connection size of each target host
type ConnRecord struct {
	ConnSize  uint            // total size of current connections
	Addresses map[string]uint // current connections as well as its count
	Writer    *io.Writer      // terminal writer  todo defer Flush
    OnChange  func(status ConnStatus)
	Mutex     *sync.Mutex
}

// connection status when a connection is added or removed.
type ConnStatus struct {
	Address string
	IsNew   bool
	Type    int
}

func NewConnRecord() *ConnRecord {
	cr := ConnRecord{ConnSize: 0, OnChange: nil}
	cr.Addresses = make(map[string]uint)
	cr.Mutex = &sync.Mutex{}
	return &cr
}

func (cr *ConnRecord) Update(status ConnStatus) {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	if status.IsNew {
		cr.ConnSize++
		if size, ok := cr.Addresses[status.Address]; ok {
			cr.Addresses[status.Address] = size + 1
		} else {
			cr.Addresses[status.Address] = 1
		}
	} else {
		cr.ConnSize--
		if size, ok := cr.Addresses[status.Address]; ok && size > 0 {
			if size-1 == 0 {
				delete(cr.Addresses, status.Address)
			} else {
				cr.Addresses[status.Address] = size - 1
			}
		} else {
			logrus.Fatal("bad connection size")
		}
	}
	// update log
	if cr.OnChange != nil {
        cr.OnChange(status)
	}
}
