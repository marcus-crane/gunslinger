package jobs

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
	"gorm.io/gorm"

	"github.com/marcus-crane/gunslinger/models"
)

var (
	CurrentPlaybackItem models.MediaItem
)

func SetupInBackground(database *gorm.DB) *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	s.Every(1).Seconds().Do(GetCurrentlyPlayingPlex, database)
	// s.Every(30).Seconds().Do(GetPlaystationPresence)
	s.Every(30).Seconds().Do(GetCurrentlyPlayingSteam, database)

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}
