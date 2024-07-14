package main

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	plexSessionEndpoint = "/status/sessions"
)

func buildPlexURL(endpoint string) string {
	plexHostURL := utils.MustEnv("PLEX_URL")
	plexToken := utils.MustEnv("PLEX_TOKEN")
	return fmt.Sprintf("%s%s?X-Plex-Token=%s", plexHostURL, endpoint, plexToken)
}

func GetCurrentlyPlayingPlex(ps *PlaybackSystem, client http.Client) {
	sessionURL := buildPlexURL(plexSessionEndpoint)
	req, err := http.NewRequest("GET", sessionURL, nil)
	if err != nil {
		slog.Error("Failed to prepare Plex request", slog.String("stack", err.Error()))
		return
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to contact Plex for updates", slog.String("stack", err.Error()))
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to parse Plex response", slog.String("stack", err.Error()))
		return
	}
	var plexResponse models.PlexResponse

	if err = json.Unmarshal(body, &plexResponse); err != nil {
		slog.Error("Error fetching Plex data", slog.String("stack", err.Error()))
	}

	// index := 0

	if plexResponse.MediaContainer.Size == 0 {
		// Nothing is playing so mark all existing items as inactive
		ps.DeactivateBySource(string(Plex))
		return
	}

	for idx, entry := range plexResponse.MediaContainer.Metadata {
		// We don't want to capture movie trailers as historical items
		if entry.Type == "clip" {
			continue
		}
		// Skip sessions that aren't from my own account
		if entry.User.Id != "1" {
			continue
		}
		mediaItem := plexResponse.MediaContainer.Metadata[idx]
		thumbnail := mediaItem.Thumb

		// Tracks generally don't have a unique cover so we should use the album cover instead
		// This should hold true even for singles though
		if mediaItem.Type == "track" {
			thumbnail = mediaItem.ParentThumb
		}

		thumbnailUrl := buildPlexURL(thumbnail)
		image, extension, domColours, err := utils.ExtractImageContent(thumbnailUrl)
		if err != nil {
			slog.Error("Failed to extract image content",
				slog.String("stack", err.Error()),
				slog.String("image_url", thumbnailUrl),
			)
			continue
		}
		imageLocation, _ := utils.BytesToGUIDLocation(image, extension)

		playingItem := models.MediaItem{
			CreatedAt:       time.Now().Unix(),
			Title:           mediaItem.Title,
			Category:        mediaItem.Type,
			Elapsed:         mediaItem.ViewOffset,
			Duration:        mediaItem.Duration,
			Source:          "plex",
			DominantColours: domColours,
			Image:           imageLocation,
		}

		if mediaItem.Player.State == "playing" {
			playingItem.IsActive = true
		}

		if mediaItem.Type == "episode" {
			playingItem.Title = fmt.Sprintf(
				"%02dx%02d %s",
				mediaItem.ParentIndex, // Season number
				mediaItem.Index,       // Episode number
				mediaItem.Title,
			)
		}

		if mediaItem.Type == "movie" {
			playingItem.Subtitle = mediaItem.Director[0].Name
		} else {
			playingItem.Subtitle = mediaItem.GrandparentTitle
		}

		// TODO: Save image one time
		// if err := saveCover(guid.String(), image, extension); err != nil {
		// 	slog.Error("Failed to save cover for Plex",
		// 		slog.String("stack", err.Error()),
		// 		slog.String("guid", guid.String()),
		// 		slog.String("title", playingItem.Title),
		// 	)
		// }
	}
}
