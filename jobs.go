package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/marcus-crane/gunslinger/utils"
)

var (
	STORAGE_DIR = utils.GetEnv("STORAGE_DIR", "/tmp")
)

func SetupInBackground(ps *PlaybackSystem) *gocron.Scheduler {
	s := gocron.NewScheduler(time.UTC)

	client := http.Client{}

	s.Every(1).Seconds().Do(GetCurrentlyPlayingPlex, ps, client)

	// If we're redeployed, we'll populate the latest state
	ps.RefreshCurrentPlayback()

	return s
}

func LoadCover(hash string, extension string) (string, error) {
	img, err := os.ReadFile(fmt.Sprintf("%s/%s.%s", STORAGE_DIR, strings.ReplaceAll(hash, ":", "."), extension))
	if err != nil {
		return "", err
	}
	return string(img), nil
}

func saveCover(hash string, image []byte, extension string) error {
	return os.WriteFile(fmt.Sprintf("%s/%s.%s", STORAGE_DIR, strings.ReplaceAll(hash, ":", "."), extension), image, 0644)
}
