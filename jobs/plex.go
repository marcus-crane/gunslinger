package jobs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/marcus-crane/gunslinger/models"
	"net/http"
	"os"
	"strconv"
)

const (
	plexSessionEndpoint = "/status/sessions"
	UserAgent           = "Gunslinger/1.0 (gunslinger@utf9k.net)"
)

func buildPlexURL(endpoint string) string {
	plexHostURL := os.Getenv("PLEX_URL")
	plexToken := os.Getenv("PLEX_TOKEN")
	return fmt.Sprintf("%s%s?X-Plex-Token=%s", plexHostURL, endpoint, plexToken)
}

func getImageBase64(thumbnailURL string) string {
	imageUrl := buildPlexURL(thumbnailURL)
	imageA := fiber.Get(imageUrl).
		UserAgent(UserAgent)
	_, body, errs := imageA.Bytes()

	if len(errs) != 0 {
		panic(errs)
	}

	var base64Encoding string

	mimeType := http.DetectContentType(body)

	switch mimeType {
	case "image/jpeg":
		base64Encoding += "data:image/jpeg;base64,"
	case "image/png":
		base64Encoding += "data:image/png;base64,"
	}

	base64Encoding += base64.StdEncoding.EncodeToString(body)

	return base64Encoding
}

func GetCurrentlyPlayingPlex() {
	sessionURL := buildPlexURL(plexSessionEndpoint)
	sessionA := fiber.Get(sessionURL).
		UserAgent(UserAgent).
		Add("Accept", "application/json").
		Add("Content-Type", "application/json")

	var plexResponse models.PlexResponse

	_, body, errs := sessionA.Bytes()

	if len(errs) != 0 {
		panic(errs)
	}

	err := json.Unmarshal(body, &plexResponse)

	if err != nil {
		fmt.Println("Error fetching Plex data: ", err)
	}

	if len(plexResponse.MediaContainer.Metadata) == 0 {
		return
	}

	index := 0

	if len(plexResponse.MediaContainer.Metadata) > 1 {
		containsPlayingItem := false
		for idx, entry := range plexResponse.MediaContainer.Metadata {
			if entry.Player.State == "playing" {
				containsPlayingItem = true
				index = idx
			}
		}
		if !containsPlayingItem {
			return
		}
	}

	mediaItem := plexResponse.MediaContainer.Metadata[index]

	duration, err := strconv.Atoi(mediaItem.Duration)
	if err != nil {
		panic(err)
	}

	elapsed, err := strconv.Atoi(mediaItem.ViewOffset)
	if err != nil {
		panic(err)
	}

	playingItem := models.MediaItem{
		Title:    mediaItem.Title,
		Category: mediaItem.Type,
		Elapsed:  elapsed,
		Duration: duration,
		Image:    getImageBase64(mediaItem.Thumb),
	}

	if mediaItem.Player.State == "playing" {
		playingItem.IsActive = true
	}

	if mediaItem.Type == "movie" {
		playingItem.Subtitle = mediaItem.Director[0].Name
	} else {
		playingItem.Subtitle = mediaItem.GrandparentTitle
	}

	CurrentPlaybackItem = playingItem
}
