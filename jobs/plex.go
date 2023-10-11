package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/r3labs/sse/v2"

	"github.com/marcus-crane/gunslinger/events"
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

func GetCurrentlyPlayingPlex(database *sqlx.DB, client http.Client) {
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

	index := 0

	containsPlayingItem := false
	if plexResponse.MediaContainer.Size > 0 {
		for idx, entry := range plexResponse.MediaContainer.Metadata {
			// We don't want to capture movie trailers as historical items
			if entry.Type == "clip" {
				continue
			}
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
			// reflect.DeepEqual is good enough for our purposes even though
			// it doesn't do things like properly copmare timestamp metadata.
			// For just checking if we should emit a message, it's good enough
			byteStream := new(bytes.Buffer)
			json.NewEncoder(byteStream).Encode(CurrentPlaybackItem)
			events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		}
		return
	}

	mediaItem := plexResponse.MediaContainer.Metadata[index]
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
		return
	}

	imageLocation, guid := utils.BytesToGUIDLocation(image, extension)

	playingItem := models.MediaItem{
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

	// reflect.DeepEqual is good enough for our purposes even though
	// it doesn't do things like properly copmare timestamp metadata.
	// For just checking if we should emit a message, it's good enough
	if !reflect.DeepEqual(CurrentPlaybackItem, playingItem) {
		byteStream := new(bytes.Buffer)
		json.NewEncoder(byteStream).Encode(playingItem)
		events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		// We want to make sure that we don't resave if the server restarts
		// to ensure the history endpoint is relatively accurate
		var previousItem models.ComboDBMediaItem
		if err := database.Get(
			&previousItem,
			"SELECT * FROM db_media_items WHERE category = ? ORDER BY created_at desc LIMIT 1",
			playingItem.Category,
		); err == nil || err.Error() == "sql: no rows in result set" {
			if CurrentPlaybackItem.Title != playingItem.Title && previousItem.Title != playingItem.Title {
				if err := saveCover(guid.String(), image, extension); err != nil {
					slog.Error("Failed to save cover for Plex",
						slog.String("stack", err.Error()),
						slog.String("guid", guid.String()),
						slog.String("title", playingItem.Title),
					)
				}
				schema := `INSERT INTO db_media_items (created_at, title, subtitle, category, is_active, duration_ms, dominant_colours, source, image) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
				_, err := database.Exec(
					schema,
					time.Now().Unix(),
					playingItem.Title,
					playingItem.Subtitle,
					playingItem.Category,
					playingItem.IsActive,
					playingItem.Duration,
					playingItem.DominantColours,
					playingItem.Source,
					playingItem.Image,
				)
				if err != nil {
					slog.Error("Failed to save DB entry",
						slog.String("stack", err.Error()),
						slog.String("title", playingItem.Title),
					)
				}
			}
		} else {
			slog.Error("An unknown error occurred",
				slog.String("stack", err.Error()),
				slog.String("title", playingItem.Title),
			)
		}
	}

	CurrentPlaybackItem = playingItem
}
