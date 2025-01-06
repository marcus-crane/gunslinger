package trakt

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/utils"
)

var (
	traktListeningEndpoint = "https://api.trakt.tv/users/sentry/listening?extended=full"
)

func getArtFromApple(traktResponse NowPlayingResponse) (string, error) {
	url := fmt.Sprintf("https://podcasts.apple.com/us/podcast/%s/id%d", traktResponse.Podcast.IDs.Slug, traktResponse.Podcast.IDs.Apple)
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}

	var coverUrl string

	// Find the review items
	doc.Find("source[type='image/jpeg']").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			srcset, exists := s.Attr("srcset")
			if !exists {
				return
			}
			bits := strings.Split(srcset, " ")
			coverUrl = strings.Replace(bits[0], "268x0", "512x0", 1)
		}
	})

	return coverUrl, nil
}

func GetCurrentlyListening(cfg config.Config, ps *playback.PlaybackSystem, client http.Client) {
	req, err := http.NewRequest("HEAD", traktListeningEndpoint, nil)
	if err != nil {
		slog.Error("Failed to build HEAD request for Traktcasts", slog.String("error", err.Error()))
		return
	}
	req.Header = http.Header{
		"Accept":            []string{"application/json"},
		"Authorization":     []string{fmt.Sprintf("Bearer %s", cfg.Trakt.BearerToken)},
		"Content-Type":      []string{"application/json"},
		"trakt-api-version": []string{"2"},
		"trakt-api-key":     []string{cfg.Trakt.ClientId},
		"User-Agent":        []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to make HEAD request to Traktcasts",
			slog.String("error", err.Error()),
			slog.String("code", res.Status),
		)
		return
	}

	// Nothing is playing so we should check if anything needs to be cleaned up
	// or if we need to do a state transition
	if res.StatusCode == 204 {
		ps.DeactivateBySource(string(playback.TraktCasts))
		return
	}

	// Do a proper, more expensive request now that we've got something fresh
	req2, err := http.NewRequest("GET", traktListeningEndpoint, nil)
	if err != nil {
		slog.Error("Failed to build GET request for Traktcasts", slog.String("error", err.Error()))
		return
	}
	req2.Header = req.Header
	res2, err := client.Do(req2)
	if err != nil {
		slog.Error("Failed to make GET request for Traktcasts", slog.String("error", err.Error()))
		return
	}

	// Nothing is playing so we should check if anything needs to be cleaned up
	// or if we need to do a state transition
	if res2.StatusCode == 204 {
		ps.DeactivateBySource(string(playback.TraktCasts))
		return
	}

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		slog.Error("Failed to read Traktcasts response",
			slog.String("error", err.Error()),
			slog.String("code", res2.Status),
		)
		return
	}
	var traktResponse NowPlayingResponse

	if err = json.Unmarshal(body, &traktResponse); err != nil {
		// TODO: Check status code
		slog.Error("Failed to unmarshal Traktcasts response",
			slog.String("error", err.Error()),
			slog.String("code", res2.Status),
		)
		return
	}

	// If we don't have any IDs, we can't look up cover art so we'll just bail out
	if traktResponse.Podcast.IDs.Apple == 0 || traktResponse.Podcast.IDs.Slug == "" {
		slog.Info("No IDs")
		return
	}

	imageUrl, err := getArtFromApple(traktResponse)
	if err != nil {
		slog.Error("Failed to retrieve art from Apple Podcasts",
			slog.String("error", err.Error()),
		)
		return
	}
	if imageUrl == "" {
		slog.Error("Did not find a suitable cover art from Apple Podcasts")
		return
	}
	image, extension, domColours, err := utils.ExtractImageContent(imageUrl)
	if err != nil {
		slog.Error("Failed to extract image content",
			slog.String("error", err.Error()),
			slog.String("image_url", imageUrl),
		)
		return
	}
	imageLocation, _ := utils.BytesToGUIDLocation(image, extension)

	// We may pause and restart episodes so we need to infer the current progress by taking the runtime
	// and checking the difference between now and when the scrobble expires
	ends, err := time.Parse("2006-01-02T15:04:05.999Z", traktResponse.ExpiresAt)
	if err != nil {
		slog.Error("Failed to parse Traktcasts expiry time",
			slog.String("error", err.Error()),
			slog.String("expires_at", traktResponse.ExpiresAt),
		)
		return
	}

	duration := traktResponse.PodcastEpisode.Runtime * 1000
	elapsed := duration - int(time.Until(ends).Milliseconds())

	update := playback.Update{
		MediaItem: playback.MediaItem{
			Title:           traktResponse.PodcastEpisode.Title,
			Subtitle:        traktResponse.Podcast.Title,
			Category:        traktResponse.Type,
			Duration:        duration,
			Source:          string(playback.Trakt),
			Image:           imageLocation,
			DominantColours: domColours,
		},
		Elapsed: time.Duration(elapsed) * time.Millisecond,
		Status:  playback.StatusPlaying,
	}

	if err := ps.UpdatePlaybackState(update); err != nil {
		slog.Error("Failed to save Steam update",
			slog.String("error", err.Error()),
			slog.String("title", update.MediaItem.Title))
	}

	hash := playback.GenerateMediaID(&update)
	if err := utils.SaveCover(cfg, hash, image, extension); err != nil {
		slog.Error("Failed to save cover for Steam",
			slog.String("error", err.Error()),
			slog.String("guid", hash),
			slog.String("title", update.MediaItem.Title),
		)
	}
}
