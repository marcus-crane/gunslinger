package events

import "github.com/r3labs/sse/v2"

var Server *sse.Server

func Init() {
	server := sse.New()
	server.AutoReplay = false
	Server = server
}
