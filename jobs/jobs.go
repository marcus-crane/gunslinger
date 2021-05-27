package jobs

import (
  "time"

  "github.com/go-co-op/gocron"
)

func BackgroundSetup() {
  s := gocron.NewScheduler(time.UTC)

  s.Every(5).Seconds().Do(Hello)

  s.StartAsync()
}
