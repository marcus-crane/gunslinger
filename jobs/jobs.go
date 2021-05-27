package jobs

import (
  "time"

  "github.com/go-co-op/gocron"
)

func BackgroundSetup() {
  s := gocron.NewScheduler(time.UTC)

  s.Every(3601).Seconds().Do(RefreshAccessToken)

  s.StartAsync()
}
