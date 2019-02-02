package ticker

import (
	"github.com/segmentio/ksuid"
	"log"
	"sync"
	"time"
)


// key of Ticker map
type TickId ksuid.KSUID

type Ticker struct {
	tickMap map[TickId]func()
	mutex   sync.RWMutex
	done    chan struct{}
}

// create a new Ticker.
func NewTicker() *Ticker {
	var t Ticker
	t.tickMap = make(map[TickId]func())
	return &t
}

// start the Ticker until receiving 'done' message
func (t *Ticker) Start() {
	go func() {
		ticker := time.NewTicker(time.Microsecond * time.Duration(100))
		defer ticker.Stop()
		for {
			select {
			case <-t.done:
				log.Println("Ticker done.")
				return
			case <-ticker.C:
				t.mutex.RLock()
				for _, tick := range t.tickMap {
					tick()
				}
				t.mutex.RUnlock()
			}
		}
	}()
}

// stop Ticker
func (t *Ticker) Stop() {
	t.done <- struct{}{}
}

func (t *Ticker) Append(id TickId, ticker func()) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tickMap[id] = ticker
}

func (t *Ticker) Remove(id TickId) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.tickMap, id)
}
