package wss

import (
	"fmt"
	"io"
	"testing"
	"time"
)

func TestBufferWRClose(t *testing.T) {
	done := make(chan struct{}, 1)
	bwr := NewBufferWR()
	go func() {
		var d1 [1024]byte
		for {
			if _, err := bwr.Read(d1[:]); err == io.EOF {
				break
			}
		}
		println("read passed")
		done <- struct{}{}
	}()

	_ = bwr.Close()

	<-done
}

func TestBufferWRWrite(t *testing.T) {
	done := make(chan struct{}, 1)
	bwr := NewBufferWR()

	var d2 [1024]byte

	_, _ = bwr.Write(d2[:])
	_, _ = bwr.Write(d2[:])
	_, _ = bwr.Write(d2[:])

	<-done
}

func TestBufferWR(t *testing.T) {
	done := make(chan struct{}, 1)
	bwr := NewBufferWR()

	go func() {
		var d1 [3 * 1024]byte
		for i := 0; i < 1; i++ {
			if _, err := bwr.Read(d1[:]); err == io.EOF {
				break
			}
		}
		fmt.Println("read passed")
		done <- struct{}{}
	}()

	var d2 [1024]byte
	_, _ = bwr.Write(d2[:]) // 3 writes, but only one read
	_, _ = bwr.Write(d2[:])
	_, _ = bwr.Write(d2[:])

	<-done
}

func TestBufferWR2(t *testing.T) {
	done := make(chan struct{}, 1)
	bwr := NewBufferWR()

	go func() {
		var d1 [ 1024]byte
		for i := 0; i < 3; i++ {
			if _, err := bwr.Read(d1[:]); err == io.EOF {
				break
			}
		}
		fmt.Println("read passed")
		done <- struct{}{}
	}()

	var d2 [1024]byte
	_, _ = bwr.Write(d2[:]) // 3 writes, with 3 reads
	time.Sleep(1 * time.Second)
	_, _ = bwr.Write(d2[:])
	time.Sleep(1 * time.Second)
	_, _ = bwr.Write(d2[:])

	<-done
}
