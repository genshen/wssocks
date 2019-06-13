package term_view

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"sync"
)

type Status struct {
	Address string
	IsNew   bool
}
type ProgressLog struct {
	ConnSize uint            // size of current connections
	Address  map[string]uint // current requests as well as its count
	log      *logrus.Logger
	Writer   *Writer // terminal writer  todo defer Flush
	mtx      *sync.Mutex
}

func NewPLog() *ProgressLog {
	plog := ProgressLog{
		ConnSize: 0,
		Address:  map[string]uint{},
		mtx:      &sync.Mutex{},
	}
	plog.Writer = NewWriter()
	plog.log = logrus.New()
	plog.log.SetOutput(plog.Writer)                                 // use terminal writer
	plog.log.SetFormatter(&logrus.TextFormatter{ForceColors: true}) // use colorful log
	plog.log.SetLevel(logrus.TraceLevel)
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
			log.Fatal("bad connection size")
		}
	}

	// update log
	_, terminalRows := getTermSize()
	// log size is ok for terminal (at least one row)
	p.log.WithField("size", p.ConnSize).Trace("size of proxy connection(s).")
	terminalRows--
	recordsWritten := 0
	if terminalRows >= 2 { // at least 2 lines left: one for show more records and one for new line(\n).
		// have rows left
		for k, v := range p.Address {
			if terminalRows <= 2 {
				// hide left records
				p.Writer.Write([]byte(fmt.Sprintf("more: %d record(s) hiden.\n", len(p.Address)-recordsWritten)))
				break
			} else {
				p.log.WithFields(logrus.Fields{
					"address": k,
					"size":    v,
				}).Info("connection size to remote proxy server.")
				terminalRows--
				recordsWritten++
			}
		}
	}

	p.Writer.Flush()
}
