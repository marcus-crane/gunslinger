package main

import (
	"net/http"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/marcus-crane/gunslinger/anilist"
	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/plex"
	"github.com/marcus-crane/gunslinger/retroachievements"
	"github.com/marcus-crane/gunslinger/spotify"
	"github.com/marcus-crane/gunslinger/steam"
	"github.com/marcus-crane/gunslinger/trakt"
)

func SetupInBackground(cfg config.Config, ps *playback.PlaybackSystem, store db.Store) (gocron.Scheduler, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		return nil, err
	}

	client := http.Client{}

	go spotify.SetupSpotifyPoller(cfg, ps, store)

	s.NewJob(
		gocron.DurationJob(time.Minute),
		gocron.NewTask(retroachievements.GetCurrentlyPlaying, cfg, ps, client),
	)

	s.NewJob(
		gocron.DurationJob(time.Second),
		gocron.NewTask(plex.GetCurrentlyPlaying, cfg, ps, client),
	)

	s.NewJob(
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(anilist.GetRecentlyReadManga, cfg, ps, store, client), // Rate limit: 90 req/sec
	)

	s.NewJob(
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(steam.GetCurrentlyPlaying, cfg, ps, client),
	)

	s.NewJob(
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(trakt.GetCurrentlyPlaying, cfg, ps, client, store),
		gocron.WithSingletonMode(gocron.LimitModeReschedule), // Make sure we block while waiting for a token on startup instead of closing the server with a new job
	)

	s.NewJob(
		gocron.DurationJob(time.Hour*6),
		gocron.NewTask(trakt.CheckForTokenRefresh, cfg, store),
	)

	// If we're redeployed, we'll populate the latest state
	ps.RefreshCurrentPlayback()

	return s, nil
}
