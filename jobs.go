package main

import (
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
)

func SetupInBackground(ps *PlaybackSystem) *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	client := http.Client{}

	s.Every(1).Seconds().Do(GetCurrentlyPlayingPlex, ps, client)

	// If we're redeployed, we'll populate the latest state
	ps.RefreshCurrentPlayback()

	return s
}
