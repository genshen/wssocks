package term_view

import (
	"fmt"
	"github.com/genshen/wssocks/wss"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"text/tabwriter"
)

type ProgressLog struct {
	Writer *Writer // terminal writer  todo defer Flush
	record *wss.ConnRecord
}

func NewPLog(cr *wss.ConnRecord) *ProgressLog {
	plog := ProgressLog{
		record: cr,
	}
	plog.Writer = NewWriter()
	return &plog
}

// update progress log.
func (p *ProgressLog) SetLogBuffer(r *wss.ConnRecord) {
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

	var recordsHiden = len(r.Addresses)
	if terminalRows >= 2 { // at least 2 lines left: one for show more records and one for new line(\n).
		// have rows left
		for addr, size := range r.Addresses {
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
			_, _ = fmt.Fprintf(w, "TOTAL\t%d\t\n", r.ConnSize)
		} else {
			_, _ = w.Write([]byte(fmt.Sprintf("TOTAL\t%d\t(%d record(s) hiden)\t\n",
				r.ConnSize, recordsHiden)))
		}
	}
}

// write interface: write buffer data directly to stdout.
func (p *ProgressLog) Write(buf []byte) (int, error) {
	p.record.Mutex.Lock()
	defer p.record.Mutex.Unlock()
	p.SetLogBuffer(p.record) // call Writer.Write() to set log data into buffer
	err := p.Writer.Flush(func() error {                      // flush buffer
		if _, err := p.Writer.OutDev.Write(buf); err != nil { // just write buff to stdout, and keep progress log.
			return err
		}
		return nil
	})
	return len(buf), err
}
