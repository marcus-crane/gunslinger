package jobs

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

func SetupInBackground() *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	s.Every(3601).Seconds().Do(RefreshAccessToken)
	s.Every(5).Seconds().Do(GetCurrentlyPlaying)
	s.Every(10).Seconds().Do(GetCurrentlyPlayingMedia)

	log.Print("Jobs scheduled. Scheduler not running yet.")

	return s
}
