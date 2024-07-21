package jobs

import (
	"log"
	"net/http"
	"time"

	"github.com/go-co-op/gocron"

	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/models"
)

var (
	CurrentPlaybackItem models.MediaItem
)

func SetupInBackground(store db.Store) *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	client := http.Client{}

	// s.Every(1).Seconds().Do(plex.GetCurrentlyPlaying, store, client)
	s.Every(15).Seconds().Do(GetRecentlyReadManga, store, client) // Rate limit: 90 req/sec
	s.Every(15).Seconds().Do(GetCurrentlyPlayingSteam, store, client)
	s.Every(15).Seconds().Do(GetCurrentlyPlayingTrakt, store, client)
	s.Every(15).Seconds().Do(GetCurrentlyListeningTrakt, store, client)

	// Assuming we have just redeployed or have crashed, we will
	// attempt to preload the last seen item in memory
	result, err := store.GetNewest()
	if err == nil {
		if result.Title != "" && result.Source != "" && CurrentPlaybackItem.Title == "" && CurrentPlaybackItem.Source == "" {
			CurrentPlaybackItem = models.MediaItem{
				CreatedAt:       result.OccuredAt,
				Title:           result.Title,
				Subtitle:        result.Subtitle,
				Category:        result.Category,
				Source:          result.Source,
				IsActive:        false,
				Backfilled:      true,
				DominantColours: result.DominantColours,
				Image:           result.Image,
			}
		}
	}

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}
