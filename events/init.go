package events

import "github.com/r3labs/sse/v2"

var (
	Server         *sse.Server
	SessionsSeen   uint64
	ActiveSessions uint64
)

type Sessions struct {
	SessionsSeen   uint64 `json:"sessions_seen"`
	ActiveSessions uint64 `json:"active_sessions"`
}

func Init() {
	server := sse.New()
	server.AutoReplay = false
	server.OnSubscribe = countSession
	server.OnUnsubscribe = sessionClosed
	Server = server
}

func countSession(streamID string, sub *sse.Subscriber) {
	SessionsSeen += 1
	ActiveSessions += 1
}

func sessionClosed(streamID string, sub *sse.Subscriber) {
	ActiveSessions -= 1
}
