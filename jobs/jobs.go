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
	s.Every(30).Seconds().Do(GetCurrentlyPlayingSteam, database)

	// Assuming we have just redeployed or have crashed, we will
	// attempt to preload the last seen item in memory
	var result models.DBMediaItem
	database.Limit(1).Order("created_at desc").Find(&result)
	if result.Title != "" && result.Source != "" && CurrentPlaybackItem.Title == "" && CurrentPlaybackItem.Source == "" {
		CurrentPlaybackItem = models.MediaItem{
			Title:      result.Title,
			Subtitle:   result.Subtitle,
			Category:   result.Category,
			Source:     result.Source,
			IsActive:   false,
			Backfilled: true,
		}
	}

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}
