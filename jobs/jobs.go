package jobs

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"gorm.io/gorm"

	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
)

var (
	CurrentPlaybackItem models.MediaItem
	STORAGE_DIR         = utils.GetEnv("STORAGE_DIR", "/tmp")
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
			Image:      loadCover(result.Category),
		}
	}

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}

func loadCover(category string) string {
	img, err := os.ReadFile(fmt.Sprintf("%s/cover.%s", STORAGE_DIR, category))
	if err != nil {
		return "https://picsum.photos/204?blur=2"
	}
	return string(img)
}

func saveCover(cover string, category string) error {
	return os.WriteFile(fmt.Sprintf("%s/cover.%s", STORAGE_DIR, category), []byte(cover), 0644)
}
