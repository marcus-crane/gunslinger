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

	index := 0

	containsPlayingItem := false
	if plexResponse.MediaContainer.Size > 0 {
		for idx, entry := range plexResponse.MediaContainer.Metadata {
			if entry.Player.State == "playing" {
				containsPlayingItem = true
				// We may have multiple items in our queue at once
				// For example, a paused song while watching a TV show
				// so we need to figure out which item (if any) is the one
				// to surface
				index = idx
			}
		}
	}
	if !containsPlayingItem {
		// We may have removed the item entirely from the play queue so it won't
		// be in the API but we know if the source is Plex and nothing in Plex
		// is playing (it would be in the API if it were) then it's safe to
		// mark it as inactive
		if CurrentPlaybackItem.IsActive && CurrentPlaybackItem.Source == "plex" {
			CurrentPlaybackItem.IsActive = false
		}
		return
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

	thumbnail := mediaItem.Thumb

	// Tracks generally don't have a unique cover so we should use the album cover instead
	// This should hold true even for singles though
	if mediaItem.Type == "track" {
		thumbnail = mediaItem.ParentThumb
	}

	playingItem := models.MediaItem{
		Title:    mediaItem.Title,
		Category: mediaItem.Type,
		Elapsed:  elapsed,
		Duration: duration,
		Source:   "plex",
		// TODO: Make use of the transcode endpoint or pull the thumbnail onto disc for caching
		Image: getImageBase64(thumbnail),
	}

	if mediaItem.Player.State == "playing" {
		playingItem.IsActive = true
	}

	if mediaItem.Type == "episode" {
		seasonNumber, err := strconv.Atoi(mediaItem.ParentIndex)
		if err != nil {
			panic(err)
		}
		episodeNumber, err := strconv.Atoi(mediaItem.Index)
		if err != nil {
			panic(err)
		}
		playingItem.Title = fmt.Sprintf(
			"%02dx%02d %s",
			seasonNumber,
			episodeNumber,
			mediaItem.Title,
		)
	}

	if mediaItem.Type == "movie" {
		playingItem.Subtitle = mediaItem.Director[0].Name
	} else {
		playingItem.Subtitle = mediaItem.GrandparentTitle
	}

	CurrentPlaybackItem = playingItem
}
