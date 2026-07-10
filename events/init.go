package events

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/r3labs/sse/v2"
)

var (
	Server         *sse.Server
	SessionsSeen   atomic.Int64
	ActiveSessions atomic.Int64
)

type Sessions struct {
	SessionsSeen   int64 `json:"sessions_seen"`
	ActiveSessions int64 `json:"active_sessions"`
}

func Init() {
	server := sse.New()
	server.AutoReplay = false
	server.OnSubscribe = countSession
	server.OnUnsubscribe = sessionClosed
	Server = server
}

func countSession(streamID string, sub *sse.Subscriber) {
	SessionsSeen.Add(1)
	ActiveSessions.Add(1)
}

func sessionClosed(streamID string, sub *sse.Subscriber) {
	ActiveSessions.Add(-1)
}

func ServeSSE(w http.ResponseWriter, r *http.Request) {
	dw := &deadlineWriter{w: w, rc: http.NewResponseController(w)}
	Server.ServeHTTP(dw, r)
}

type deadlineWriter struct {
	w  http.ResponseWriter
	rc *http.ResponseController
}

func (d *deadlineWriter) Header() http.Header { return d.w.Header() }

func (d *deadlineWriter) WriteHeader(statusCode int) { d.w.WriteHeader(statusCode) }

func (d *deadlineWriter) Write(b []byte) (int, error) {
	_ = d.rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
	return d.w.Write(b)
}

func (d *deadlineWriter) Flush() {
	_ = d.rc.Flush()
}
