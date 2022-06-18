package jobs

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"

	"github.com/marcus-crane/gunslinger/models"
)

var (
	CurrentPlaybackItem models.MediaItem
)

func SetupInBackground() *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	s.Every(1).Seconds().Do(GetCurrentlyPlayingPlex)
	// s.Every(30).Seconds().Do(GetPlaystationPresence)
	s.Every(5).Seconds().Do(GetCurrentlyPlayingSteam)

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}
