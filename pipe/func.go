package pipe

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// 是否打开调试日志
var pipeDebug bool = true

// 数据过期时间
var expHour time.Duration = time.Duration(1) * time.Hour
var expTenMinute time.Duration = time.Duration(10) * time.Minute

// 状态值
const (
	StaWait  = "wait"
	StaSend  = "send"
	StaClose = "close"
)

type PipeWriter interface {
	Write(p []byte) (n int, err error)
	WriteEOF()
}

type buffer struct {
	eof  bool
	data []byte
}

// CopyBuffer 传输数据
func CopyBuffer(iow PipeWriter, conn *net.TCPConn) (written int64, err error) {
	//如果设置过大会耗内存高，4k比较合理
	size := 4 * 1024
	if pipeDebug {
		size = 10 //临时测试
	}
	buf := make([]byte, size)
	i := 0
	for {
		i++
		nr, er := conn.Read(buf)
		if nr > 0 {
			//fmt.Println("copy read", nr)
			var nw int
			var ew error
			nw, ew = iow.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = fmt.Errorf("#1 %s", ew.Error())
				break
			}
			if nr != nw {
				err = fmt.Errorf("#2 %s", io.ErrShortWrite.Error())
				break
			}
		}
		if er == io.EOF {
			//fmt.Println(time.Now(), "copy get and write eof")
			iow.WriteEOF()
			break
		} else if er != nil {
			err = fmt.Errorf("#3 %s", er.Error())
			break
		}
	}
	return written, err
}

// 带有超时的读
func readWithTimeout(b chan buffer, exp time.Duration) (buffer, error) {
	for {
		select {
		case <-time.After(exp):
			return buffer{}, errors.New("time out")
		case data := <-b:
			return data, nil
		}
	}
}

// 创建缓冲区
func makeBuffer() chan buffer {
	if pipeDebug {
		return make(chan buffer, 1)
	}
	return make(chan buffer, 10)
}

// 打印日志
func pipePrintln(a ...interface{}) (n int, err error) {
	if !pipeDebug {
		return 0, nil
	}
	return fmt.Println(a...)
}
