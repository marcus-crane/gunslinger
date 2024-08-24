package main

import (
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/plex"
	"github.com/marcus-crane/gunslinger/spotify"
	"github.com/marcus-crane/gunslinger/steam"
	"github.com/marcus-crane/gunslinger/trakt"
)

func SetupInBackground(cfg config.Config, ps *playback.PlaybackSystem, store db.Store) *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	client := http.Client{}

	go spotify.SetupSpotifyPoller(cfg, ps, store)

	s.Every(1).Seconds().Do(plex.GetCurrentlyPlaying, cfg, ps, client)
	// s.Every(15).Seconds().Do(anilist.GetRecentlyReadManga, ps, store, client) // Rate limit: 90 req/sec
	s.Every(15).Seconds().Do(steam.GetCurrentlyPlaying, cfg, ps, client)
	s.Every(15).Seconds().Do(trakt.GetCurrentlyPlaying, cfg, ps, client)
	s.Every(15).Seconds().Do(trakt.GetCurrentlyListening, cfg, ps, client)

	// If we're redeployed, we'll populate the latest state
	ps.RefreshCurrentPlayback()

	return s
}
