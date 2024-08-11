package trakt

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/utils"
)

var (
	traktPlayingEndpoint = "https://api.trakt.tv/users/sentry/watching"
	tmdbMovieEndpoint    = "https://api.themoviedb.org/3/movie/%d/images"
	tmdbEpisodeEndpoint  = "https://api.themoviedb.org/3/tv/%d/season/%d/episode/%d/images"
)

type NowPlayingResponse struct {
	ExpiresAt      string       `json:"expires_at"`
	StartedAt      string       `json:"started_at"`
	Action         string       `json:"action"`
	Type           string       `json:"type"`
	Movie          TraktSummary `json:"movie"`
	Episode        TraktEpisode `json:"episode"`
	Show           TraktSummary `json:"show"`
	PodcastEpisode TraktEpisode `json:"podcast_episode"`
	Podcast        TraktSummary `json:"podcast"`
}

type TraktEpisode struct {
	Season        int      `json:"season"`
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	IDs           TraktIDs `json:"ids"`
	Overview      string   `json:"overview"`
	OverviewPlain string   `json:"overview_plain"`
	Explicit      bool     `json:"explicit"`
	Runtime       int      `json:"runtime"`
}

type TraktSummary struct {
	Title         string   `json:"title"`
	Year          int      `json:"year"`
	IDs           TraktIDs `json:"ids"`
	Overview      string   `json:"overview"`
	OverviewPlain string   `json:"overview_plain"`
	Author        string   `json:"author"`
	Homepage      string   `json:"homepage"`
}

type TraktIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	TVDB  int    `json:"tvdb"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
	Apple int    `json:"apple"`
}

type TMDBImageResponse struct {
	ID      int         `json:"id"`
	Stills  []TMDBImage `json:"stills"`
	Posters []TMDBImage `json:"posters"`
}

type TMDBImage struct {
	FilePath string `json:"file_path"`
}

func getArtFromTMDB(apiKey string, traktResponse NowPlayingResponse) (string, error) {
	url := ""
	if traktResponse.Type == "movie" {
		url = fmt.Sprintf(tmdbMovieEndpoint, traktResponse.Movie.IDs.TMDB)
	} else {
		url = fmt.Sprintf(tmdbEpisodeEndpoint, traktResponse.Show.IDs.TMDB, traktResponse.Episode.Season, traktResponse.Episode.Number)
	}

	url = fmt.Sprintf("%s?api_key=%s", url, apiKey)

	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var tmdbImageResponse TMDBImageResponse

	if err = json.Unmarshal(body, &tmdbImageResponse); err != nil {
		slog.Error("Failed to fetch image data from TMDB", slog.String("body", string(body)))
	}

	imagePath := ""
	if traktResponse.Type == "movie" {
		imagePath = tmdbImageResponse.Posters[0].FilePath
	} else {
		imagePath = tmdbImageResponse.Stills[0].FilePath
	}

	return fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", imagePath), nil
}

func GetCurrentlyPlaying(ps *playback.PlaybackSystem, client http.Client) {
	traktBearerToken := utils.MustEnv("TRAKT_BEARER_TOKEN")
	traktClientID := utils.MustEnv("TRAKT_CLIENT_ID")
	tmdbToken := utils.MustEnv("TMDB_TOKEN")

	req, err := http.NewRequest("HEAD", traktPlayingEndpoint, nil)
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

	// Nothing is playing so we don't need to do any further work
	if res.StatusCode == 204 {
		ps.DeactivateBySource(string(playback.Trakt))
		return
	}

	// Do a proper, more expensive request now that we've got something fresh
	req2, err := http.NewRequest("GET", traktPlayingEndpoint, nil)
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
	if res2.StatusCode == 204 {
		ps.DeactivateBySource(string(playback.Trakt))
		return
	}

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		slog.Error("Failed to read Trakt response",
			slog.String("stack", err.Error()),
			slog.String("code", res2.Status),
		)
		return
	}
	var traktResponse NowPlayingResponse

	if err = json.Unmarshal(body, &traktResponse); err != nil {
		// TODO: Check status code
		slog.Error("Failed to unmarshal Trakt response",
			slog.String("stack", err.Error()),
			slog.String("code", res2.Status),
		)
		return
	}

	// We only want to use Trakt to capture movies and TV series that have been
	// manually checked into, to supplement that they aren't automatically being
	// captured by Plex polling. It's also wasteful to call out to TMDB and what
	// not if nothing is happening
	//
	// TODO: Handle checking for 204 and cleaning up any active resources
	if traktResponse.Action != "checkin" {
		return
	}

	imageUrl, err := getArtFromTMDB(tmdbToken, traktResponse)
	if err != nil {
		slog.Error("Failed to retrieve art from TMDB",
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
	imageLocation, _ := utils.BytesToGUIDLocation(image, extension)

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

	update := playback.Update{
		MediaItem: playback.MediaItem{
			Category:        traktResponse.Type,
			Duration:        duration,
			Source:          string(playback.Trakt),
			Image:           imageLocation,
			DominantColours: domColours,
		},
		Elapsed: time.Since(started),
		Status:  playback.StatusPlaying,
	}

	if traktResponse.Type == "movie" {
		update.MediaItem.Title = traktResponse.Movie.Title
		update.MediaItem.Subtitle = fmt.Sprint(traktResponse.Movie.Year)
	} else {
		update.MediaItem.Title = fmt.Sprintf(
			"%02dx%02d %s",
			traktResponse.Episode.Season, // Season number
			traktResponse.Episode.Number, // Episode number
			traktResponse.Episode.Title,
		)
		update.MediaItem.Subtitle = traktResponse.Show.Title
	}

	if err := ps.UpdatePlaybackState(update); err != nil {
		slog.Error("Failed to save Steam update",
			slog.String("stack", err.Error()),
			slog.String("title", update.MediaItem.Title))
	}

	hash := playback.GenerateMediaID(&update)
	if err := utils.SaveCover(hash, image, extension); err != nil {
		slog.Error("Failed to save cover for Steam",
			slog.String("stack", err.Error()),
			slog.String("guid", hash),
			slog.String("title", update.MediaItem.Title),
		)
	}
}
