package term_view

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"sync"
	"text/tabwriter"
)

type Status struct {
	Address string
	IsNew   bool
	Type    int
}

type ProgressLog struct {
	ConnSize uint            // size of current connections
	Address  map[string]uint // current requests as well as its count
	Writer   *Writer         // terminal writer  todo defer Flush
	mtx      *sync.Mutex
}

func NewPLog() *ProgressLog {
	plog := ProgressLog{
		ConnSize: 0,
		Address:  map[string]uint{},
		mtx:      &sync.Mutex{},
	}
	plog.Writer = NewWriter()
	return &plog
}

func (p *ProgressLog) Update(status Status) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if status.IsNew {
		p.ConnSize++
		if size, ok := p.Address[status.Address]; ok {
			p.Address[status.Address] = size + 1
		} else {
			p.Address[status.Address] = 1
		}
	} else {
		p.ConnSize--
		if size, ok := p.Address[status.Address]; ok && size > 0 {
			if size-1 == 0 {
				delete(p.Address, status.Address);
			} else {
				p.Address[status.Address] = size - 1
			}
		} else {
			logrus.Fatal("bad connection size")
		}
	}
	// update log
	p.setLogBuffer()    // call Writer.Write() to set log data into buffer
	p.Writer.Flush(nil) // flush buffer
}

// update progress log.
func (p *ProgressLog) setLogBuffer() {
	_, terminalRows, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		logrus.Error(err)
		return
	}
	// log size is ok for terminal (at least one row)
	w := new(tabwriter.Writer)

	w.Init(p.Writer, 0, 0, 5, ' ', 0)
	defer w.Flush()

	_, _ = fmt.Fprintf(w, "TARGETs\tCONNECTIONs\t\n")
	terminalRows--

	var recordsHiden = len(p.Address)
	if terminalRows >= 2 { // at least 2 lines left: one for show more records and one for new line(\n).
		// have rows left
		for addr, size := range p.Address {
			if terminalRows <= 2 {
				// hide left records
				break
			} else {
				_, _ = fmt.Fprintf(w, "%s\t%d\t\n", addr, size)
				terminalRows--
				recordsHiden--
			}
		}
		// log total connection size.
		if recordsHiden == 0 {
			_, _ = fmt.Fprintf(w, "TOTAL\t%d\t\n", p.ConnSize)
		} else {
			_, _ = w.Write([]byte(fmt.Sprintf("TOTAL\t%d\t(%d record(s) hiden)\t\n",
				p.ConnSize, recordsHiden)))
		}
	}
}

// write buffer data directly to stdout.
func (p *ProgressLog) Write(buf [] byte) (int, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.setLogBuffer() // call Writer.Write() to set log data into buffer
	err := p.Writer.Flush(func() error { // flush buffer
		if _, err := p.Writer.OutDev.Write(buf); err != nil { // just write buff to stdout, and keep progress log.
			return err
		}
		return nil
	})
	return len(buf), err
}
