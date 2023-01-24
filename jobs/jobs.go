package jobs

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jmoiron/sqlx"

	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
)

var (
	CurrentPlaybackItem models.MediaItem
	STORAGE_DIR         = utils.GetEnv("STORAGE_DIR", "/tmp")
)

func SetupInBackground(database *sqlx.DB) *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	s.Every(1).Seconds().Do(GetCurrentlyPlayingPlex, database)
	s.Every(30).Seconds().Do(GetCurrentlyPlayingSteam, database)

	// Assuming we have just redeployed or have crashed, we will
	// attempt to preload the last seen item in memory
	var result models.DBMediaItem
	if err := database.Get(&result, "SELECT * FROM db_media_items ORDER BY created_at desc LIMIT 1"); err == nil {
		if result.Title != "" && result.Source != "" && CurrentPlaybackItem.Title == "" && CurrentPlaybackItem.Source == "" {
			CurrentPlaybackItem = models.MediaItem{
				Title:      result.Title,
				Subtitle:   result.Subtitle,
				Category:   result.Category,
				Source:     result.Source,
				IsActive:   false,
				Backfilled: true,
				Image:      result.Image,
			}
		}
	}

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}

func LoadCover(guid string, extension string) (string, error) {
	img, err := os.ReadFile(fmt.Sprintf("%s/cover.%s.%s", STORAGE_DIR, guid, extension))
	if err != nil {
		return "", err
	}
	return string(img), nil
}

func saveCover(guid string, image []byte, extension string) error {
	return os.WriteFile(fmt.Sprintf("%s/cover.%s.%s", STORAGE_DIR, guid, extension), image, 0644)
}
