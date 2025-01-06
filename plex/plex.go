package plex

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	plexSessionEndpoint = "/status/sessions"
)

type PlexResponse struct {
	MediaContainer MediaContainer `json:"MediaContainer"`
}

type MediaContainer struct {
	Size     int        `json:"size"`
	Metadata []Metadata `json:"Metadata"`
}

type Metadata struct {
	Attribution         string     `json:"attribution"`
	Duration            int        `json:"duration"`
	GrandparentTitle    string     `json:"grandparentTitle"`
	LibrarySectionTitle string     `json:"librarySectionTitle"`
	Thumb               string     `json:"thumb"`
	ParentThumb         string     `json:"parentThumb"`
	Index               int        `json:"index"`
	ParentIndex         int        `json:"parentIndex"`
	Title               string     `json:"title"`
	Type                string     `json:"type"`
	ViewOffset          int        `json:"viewOffset"`
	Director            []Director `json:"Director"`
	Player              Player     `json:"Player"`
	User                User       `json:"User"`
}

type Director struct {
	Name string `json:"tag"`
}

type Player struct {
	State string `json:"state"`
}

type User struct {
	Id string `json:"id"`
}

func buildPlexURL(cfg config.Config, endpoint string) string {
	return fmt.Sprintf("%s%s?X-Plex-Token=%s", cfg.Plex.URL, endpoint, cfg.Plex.Token)
}

func GetCurrentlyPlaying(cfg config.Config, ps *playback.PlaybackSystem, client http.Client) {
	sessionURL := buildPlexURL(cfg, plexSessionEndpoint)
	req, err := http.NewRequest("GET", sessionURL, nil)
	if err != nil {
		slog.Error("Failed to prepare Plex request", slog.String("error", err.Error()))
		return
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to contact Plex for updates", slog.String("error", err.Error()))
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to parse Plex response", slog.String("error", err.Error()))
		return
	}
	var plexResponse PlexResponse

	if err = json.Unmarshal(body, &plexResponse); err != nil {
		slog.Error("Error fetching Plex data", slog.String("error", err.Error()))
	}

	// index := 0

	if plexResponse.MediaContainer.Size == 0 {
		// Nothing is playing so mark all existing items as inactive
		ps.DeactivateBySource(string(playback.Plex))
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
		// Don't surface downloaded YouTube videos since they're mostly low quality junk
		if entry.LibrarySectionTitle == "YouTube" {
			continue
		}
		mediaItem := plexResponse.MediaContainer.Metadata[idx]
		thumbnail := mediaItem.Thumb

		// Tracks generally don't have a unique cover so we should use the album cover instead
		// This should hold true even for singles though
		if mediaItem.Type == "track" {
			thumbnail = mediaItem.ParentThumb
		}

		thumbnailUrl := buildPlexURL(cfg, thumbnail)
		image, extension, domColours, err := utils.ExtractImageContent(thumbnailUrl)
		if err != nil {
			slog.Error("Failed to extract image content",
				slog.String("error", err.Error()),
				slog.String("image_url", thumbnailUrl),
			)
			continue
		}
		imageLocation, _ := utils.BytesToGUIDLocation(image, extension)

		title := mediaItem.Title

		if mediaItem.Type == "episode" {
			title = fmt.Sprintf(
				"%02dx%02d %s",
				mediaItem.ParentIndex, // Season number
				mediaItem.Index,       // Episode number
				mediaItem.Title,
			)
		}

		var subtitle string

		if mediaItem.Type == "movie" {
			subtitle = mediaItem.Director[0].Name
		} else {
			subtitle = mediaItem.GrandparentTitle
		}

		// If an item is stopped, it'll just not be here at all
		status := playback.StatusPlaying

		if mediaItem.Player.State == "paused" {
			status = playback.StatusPaused
		}

		elapsed := mediaItem.ViewOffset * int(time.Millisecond)

		update := playback.Update{
			MediaItem: playback.MediaItem{
				Title:           title,
				Subtitle:        subtitle,
				Category:        mediaItem.Type,
				Duration:        mediaItem.Duration,
				Source:          string(playback.Plex),
				Image:           imageLocation,
				DominantColours: domColours,
			},
			Elapsed: time.Duration(elapsed),
			Status:  status,
		}

		if err := ps.UpdatePlaybackState(update); err != nil {
			slog.Error("Failed to save Plex update",
				slog.String("error", err.Error()),
				slog.String("title", title))
		}

		hash := playback.GenerateMediaID(&update)
		if err := utils.SaveCover(cfg, hash, image, extension); err != nil {
			slog.Error("Failed to save cover for Plex",
				slog.String("error", err.Error()),
				slog.String("guid", hash),
				slog.String("title", update.MediaItem.Title),
			)
		}
	}
}
