package main

import (
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/plex"
	"github.com/marcus-crane/gunslinger/utils"
)

var (
	STORAGE_DIR = utils.GetEnv("STORAGE_DIR", "/tmp")
)

func SetupInBackground(ps playback.System) *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	client := http.Client{}

	s.Every(1).Seconds().Do(plex.GetCurrentlyPlaying, ps, client)

	// If we're redeployed, we'll populate the latest state
	ps.RefreshCurrentPlayback()

	return s
}
