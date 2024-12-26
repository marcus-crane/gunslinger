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
		gocron.NewTask(anilist.GetRecentlyReadManga, ps, store, client), // Rate limit: 90 req/sec
	)

	s.NewJob(
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(steam.GetCurrentlyPlaying, cfg, ps, client),
	)

	s.NewJob(
		gocron.DurationJob(time.Second*15),
		gocron.NewTask(trakt.GetCurrentlyPlaying, cfg, ps, client),
	)

	// If we're redeployed, we'll populate the latest state
	ps.RefreshCurrentPlayback()

	return s, nil
}
