package events

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	Server         = NewBroker()
	SessionsSeen   atomic.Int64
	ActiveSessions atomic.Int64
)

type Sessions struct {
	SessionsSeen   int64 `json:"sessions_seen"`
	ActiveSessions int64 `json:"active_sessions"`
}

type Broker struct {
	mu          sync.Mutex
	subscribers map[chan []byte]struct{}
}

func NewBroker() *Broker {
	return &Broker{subscribers: make(map[chan []byte]struct{})}
}

func (b *Broker) Publish(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subscribers {
		select {
		case ch <- data:
		default:
		}
	}
}

func (b *Broker) subscribe() chan []byte {
	ch := make(chan []byte, 8)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	SessionsSeen.Add(1)
	ActiveSessions.Add(1)
	return ch
}

func (b *Broker) unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	b.mu.Unlock()
	ActiveSessions.Add(-1)
}

func ServeSSE(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if err := rc.Flush(); err != nil {
		return
	}

	ch := Server.subscribe()
	defer Server.unsubscribe(ch)

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		var err error
		select {
		case <-r.Context().Done():
			return
		case data := <-ch:
			_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
		case <-heartbeat.C:
			_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
			_, err = fmt.Fprint(w, ": ping\n\n")
		}
		if err != nil {
			return
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}
