// most code in this sub-package is copy or modified from https://github.com/gosuri/uilive

package term_view

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// FdWriter is a writer with a file descriptor.
type FdWriter interface {
	io.Writer
	Fd() uintptr
}

// Writer will updates the terminal when flush is called.
type Writer struct {
	OutDev    io.Writer
	buf       bytes.Buffer
	mtx       *sync.Mutex
	lineCount int // lines have wrote to terminal.
}

// NewWriter returns a new Writer with default values
func NewWriter() *Writer {
	return &Writer{
		OutDev:    os.Stdout,
		mtx:       &sync.Mutex{},
		lineCount: 0,
	}
}

// Write write contents to the writer's io writer.
func (w *Writer) NormalWrite(buf []byte) (n int, err error) {
	w.mtx.Lock()
	w.clearLines() // clean progress lines first
	defer w.mtx.Unlock()
	return w.OutDev.Write(buf)
}

func (w *Writer) Write(buf []byte) (n int, err error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.buf.Write(buf)
}

// Flush writes to the out and resets the buffer.
func (w *Writer) Flush(onLinesCleared func() error) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	if len(w.buf.Bytes()) == 0 {
		return nil
	}
	w.clearLines()
	if onLinesCleared != nil {
		if err := onLinesCleared(); err != nil { // callback if lines is cleared.
			return err
		}
	}

	lines := 0
	var currentLine bytes.Buffer
	for _, b := range w.buf.Bytes() {
		if b == '\n' {
			lines++
			currentLine.Reset()
		} else {
			currentLine.Write([]byte{b})
			// todo windows overflow
			//if overFlowHandled && currentLine.Len() > termWidth {
			//	lines++
			//	currentLine.Reset()
			//}
		}
	}
	w.lineCount = lines
	_, err := w.OutDev.Write(w.buf.Bytes())
	w.buf.Reset()
	return err
}
