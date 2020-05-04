package wss

import (
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
            // with close, read can return
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
	bwr := NewBufferWR()

	var d2 [1024]byte

	_, _ = bwr.Write(d2[:])
	_, _ = bwr.Write(d2[:])
	_, _ = bwr.Write(d2[:])
}

func TestBufferWR(t *testing.T) {
	done := make(chan struct{}, 1)
	bwr := NewBufferWR()

	go func() {
		var d1 [3 * 1024]byte
		for i := 0; i < 1; i++ {
            // read all data at once
            if n, err := bwr.Read(d1[:]); err == io.EOF {
				break
            } else {
                if n != 3*1024 {
                    t.Error("read data length not match")
                }
			}
		}
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
        var d1 [1024]byte
		for i := 0; i < 3; i++ {
            if n, err := bwr.Read(d1[:]); err == io.EOF {
				break
            } else {
                if n != 1024 {
                    t.Error("read data length not match")
                }
			}
		}
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

func TestBufferWR3(t *testing.T) {
    done := make(chan struct{}, 1)
    bwr := NewBufferWR()

    go func() {
        var d1 [1024]byte
        for i := 0; i < 3; i++ {
            if n, err := bwr.Read(d1[:]); err == io.EOF {
                break
            } else {
                if n != 1024 {
                    t.Error("read data length not match")
                }
            }
        }
        done <- struct{}{}
    }()

    var d2 [3 * 1024]byte
    _, _ = bwr.Write(d2[:]) // 1 writes, but 3 reads, due to the small read buffer

    <-done
}
