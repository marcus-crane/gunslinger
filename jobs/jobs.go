package jobs

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"

	"github.com/marcus-crane/gunslinger/models"
)

var (
	CurrentPlaybackItem     models.MediaItem
	CurrentPlaybackItemV3   models.MediaItem
	CurrentPlaybackProgress models.MediaProgress
)

func SetupInBackground() *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	s.Every(3601).Seconds().Do(RefreshAccessToken)
	s.Every(3).Seconds().Do(GetCurrentlyPlaying)
	s.Every(10).Seconds().Do(GetCurrentlyPlayingMedia)
	s.Every(10).Seconds().Do(GetCurrentlyPlayingPlex)

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}
