package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
	"github.com/r3labs/sse/v2"
)

var (
	traktListeningEndpoint = "https://api.trakt.tv/users/sentry/listening"
)

func getArtFromApple(traktResponse models.NowPlayingResponse) (string, error) {
	//url := fmt.Sprintf("https://podcasts.apple.com/us/podcast/%s/id%d", traktResponse.PodcastEpisode.IDs.Slug, traktResponse.PodcastEpisode.IDs.Apple)
	// res, err := http.Get(url)
	// if err != nil {
	// 	return "", err
	// }

	// body, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	return "", err
	// }
	// var tmdbImageResponse models.TMDBImageResponse

	// if err = json.Unmarshal(body, &tmdbImageResponse); err != nil {
	// 	slog.Error("Failed to fetch image data from TMDB", slog.String("body", string(body)))
	// }

	return "https://is1-ssl.mzstatic.com/image/thumb/Podcasts113/v4/55/52/a3/5552a384-1ba2-60cc-eab5-e1c1d096e1ca/mza_7482651295927141170.jpg/626x0w.jpg", nil
}

func GetCurrentlyListeningTrakt(store db.Store, client http.Client) {
	traktBearerToken := utils.MustEnv("TRAKT_BEARER_TOKEN")
	traktClientID := utils.MustEnv("TRAKT_CLIENT_ID")

	req, err := http.NewRequest("HEAD", traktListeningEndpoint, nil)
	if err != nil {
		slog.Error("Failed to build HEAD request for Trakt", slog.String("stack", err.Error()))
		return
	}
	req.Header = http.Header{
		"Accept":            []string{"application/json"},
		"Authorization":     []string{fmt.Sprintf("Bearer %s", traktBearerToken)},
		"Content-Type":      []string{"application/json"},
		"trakt-api-version": []string{"2"},
		"trakt-api-key":     []string{traktClientID},
		"User-Agent":        []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to make HEAD request to Trakt",
			slog.String("stack", err.Error()),
			slog.String("code", res.Status),
		)
		return
	}

	// Nothing is playing so we should check if anything needs to be cleaned up
	// or if we need to do a state transition
	if res.StatusCode == 204 {
		if CurrentPlaybackItem.IsActive && CurrentPlaybackItem.Source == "traktcasts" {
			CurrentPlaybackItem.IsActive = false
			byteStream := new(bytes.Buffer)
			json.NewEncoder(byteStream).Encode(CurrentPlaybackItem)
			events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		}
		return
	}

	// Do a proper, more expensive request now that we've got something fresh
	req2, err := http.NewRequest("GET", traktListeningEndpoint, nil)
	if err != nil {
		slog.Error("Failed to build GET request for Trakt", slog.String("stack", err.Error()))
		return
	}
	req2.Header = req.Header
	res2, err := client.Do(req2)
	if err != nil {
		slog.Error("Failed to make GET request for Trakt", slog.String("stack", err.Error()))
		return
	}

	// Nothing is playing so we should check if anything needs to be cleaned up
	// or if we need to do a state transition
	if res.StatusCode == 204 {
		if CurrentPlaybackItem.IsActive && CurrentPlaybackItem.Source == "traktcasts" {
			CurrentPlaybackItem.IsActive = false
			byteStream := new(bytes.Buffer)
			json.NewEncoder(byteStream).Encode(CurrentPlaybackItem)
			events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		}
		return
	}

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		slog.Error("Failed to unmarshal Trakt response",
			slog.String("stack", err.Error()),
			slog.String("code", res2.Status),
		)
		return
	}
	var traktResponse models.NowPlayingResponse

	if err = json.Unmarshal(body, &traktResponse); err != nil {
		// TODO: Check status code
		slog.Error("Failed to unmarshal Trakt response",
			slog.String("stack", err.Error()),
			slog.String("code", res2.Status),
		)
		return
	}

	// If we don't have any IDs, we can't look up cover art so we'll just bail out
	if traktResponse.Movie.IDs.Apple == 0 || traktResponse.Movie.IDs.Slug == "" {
		return
	}

	imageUrl, err := getArtFromApple(traktResponse)
	if err != nil {
		slog.Error("Failed to retrieve art from Apple Podcasts",
			slog.String("stack", err.Error()),
		)
		return
	}
	image, extension, domColours, err := utils.ExtractImageContent(imageUrl)
	if err != nil {
		slog.Error("Failed to extract image content",
			slog.String("stack", err.Error()),
			slog.String("image_url", imageUrl),
		)
		return
	}
	imageLocation, guid := utils.BytesToGUIDLocation(image, extension)

	started, err := time.Parse("2006-01-02T15:04:05.999Z", traktResponse.StartedAt)
	if err != nil {
		slog.Error("Failed to parse Trakt start time",
			slog.String("stack", err.Error()),
			slog.String("started_at", traktResponse.StartedAt),
		)
		return
	}
	ends, err := time.Parse("2006-01-02T15:04:05.999Z", traktResponse.ExpiresAt)
	if err != nil {
		slog.Error("Failed to parse Trakt expiry time",
			slog.String("stack", err.Error()),
			slog.String("expires_at", traktResponse.ExpiresAt),
		)
		return
	}

	duration := int(ends.Sub(started).Milliseconds())
	elapsed := int(time.Since(started).Milliseconds())

	playingItem := models.MediaItem{
		Title:           traktResponse.PodcastEpisode.Title,
		Subtitle:        traktResponse.Podcast.Title,
		CreatedAt:       time.Now().Unix(),
		Category:        traktResponse.Type,
		IsActive:        true,
		Elapsed:         elapsed,
		Duration:        duration,
		Source:          "traktcasts",
		DominantColours: domColours,
		Image:           imageLocation,
	}

	// Fallback in case a scrobble somehow runs too long
	if time.Now().After(ends) {
		playingItem.IsActive = false
	}

	if CurrentPlaybackItem.Hash() != playingItem.Hash() {
		byteStream := new(bytes.Buffer)
		json.NewEncoder(byteStream).Encode(playingItem)
		events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		// We want to make sure that we don't resave if the server restarts
		// to ensure the history endpoint is relatively accurate
		previousItem, err := store.GetByCategory(playingItem.Category)
		if err == nil || err.Error() == "sql: no rows in result set" {
			if CurrentPlaybackItem.Title != playingItem.Title && previousItem.Title != playingItem.Title {
				if err := saveCover(guid.String(), image, extension); err != nil {
					slog.Error("Failed to save cover for Trakt",
						slog.String("stack", err.Error()),
						slog.String("guid", guid.String()),
						slog.String("title", playingItem.Title),
					)
				}
				if err := store.Insert(playingItem); err != nil {
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
