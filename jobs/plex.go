package jobs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/marcus-crane/gunslinger/models"
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

	var client http.Client
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		panic(err)
	}
	req.Header = http.Header{
		"User-Agent": []string{UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
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
	var client http.Client
	req, err := http.NewRequest("GET", sessionURL, nil)
	if err != nil {
		panic(err)
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var plexResponse models.PlexResponse

	if err = json.Unmarshal(body, &plexResponse); err != nil {
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
