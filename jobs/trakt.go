package jobs

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/marcus-crane/gunslinger/models"
	"log"
	"os"
	"time"
)

var (
	MediaPlaybackStatus models.Media
)

const (
	AbsoluteImageLink     = "https://image.tmdb.org/t/p/w500%s"
	TraktWatchingEndpoint = "https://api.trakt.tv/users/sentry/watching?extended=full"
	EpisodeImageEndpoint  = "https://api.themoviedb.org/3/tv/%d/season/%d/episode/%d/images"
	SeasonImageEndpoint   = "https://api.themoviedb.org/3/tv/%d/season/%d/images"
	ShowImageEndpoint     = "https://api.themoviedb.org/3/tv/%d/images"
	MovieImageEndpoint    = "https://api.themoviedb.org/3/movie/%d/images"
	userAgent             = "Now Playing/1.0 (utf9k.net)"
)

func getMediaImage(imageURL string) []byte {
	tmdbApiKey := os.Getenv("TMDB_API_KEY")

	imageA := fiber.Get(imageURL).
		UserAgent(userAgent).
		Add("Authorization", fmt.Sprintf("Bearer %s", tmdbApiKey))

	code, body, errs := imageA.Bytes()

	if len(errs) != 0 {
		panic(errs)
	}

	if code != 200 {
		fmt.Println("Received non-200 code when fetching image: ", code)
		return []byte{}
	}

	return body
}

func buildAbsoluteImageLink(images []models.Image) []models.Image {
	for idx, image := range images {
		images[idx].FilePath = fmt.Sprintf(AbsoluteImageLink, image.FilePath)
	}
	return images
}

func GetCurrentlyPlayingMedia() {
	clientID := os.Getenv("TRAKT_CLIENT_ID")

	playerA := fiber.Get(TraktWatchingEndpoint).
		UserAgent(UserAgent).
		Add("Content-Type", "application/json").
		Add("trakt-api-version", "2").
		Add("trakt-api-key", clientID)

	var traktResponse models.Media

	code, body, errs := playerA.Bytes()

	if len(errs) != 0 {
		panic(errs)
	}

	err := json.Unmarshal(body, &traktResponse)

	if err != nil && code != 204 {
		fmt.Println("Error fetching Trakt data: ", err)
	}

	if traktResponse.MediaType == "episode" {
		episodeLink := fmt.Sprintf(
			"https://trakt.tv/shows/%s/seasons/%d/episodes/%d",
			traktResponse.Show.IDs.Slug,
			traktResponse.Episode.SeasonNumber,
			traktResponse.Episode.EpisodeNumber,
		)
		traktResponse.Episode.Link = episodeLink

		showLink := fmt.Sprintf("https://trakt.tv/shows/%s", traktResponse.Show.IDs.Slug)
		traktResponse.Show.Link = showLink
	}

	if traktResponse.MediaType == "movie" {
		movieLink := fmt.Sprintf("https://www.imdb.com/title/%s/", traktResponse.Movie.IDs.IMDB)
		traktResponse.Movie.Link = movieLink
	}

	MediaPlaybackStatus = traktResponse

	if MediaPlaybackStatus.StartedAt == "" &&
		(CurrentPlaybackItem.Category == "movie" || CurrentPlaybackItem.Category == "tv") {
		CurrentPlaybackItem.IsActive = false
		CurrentPlaybackProgress.IsActive = false
		return // Nothing playing but we want to retain the last playing item
	}

	if MediaPlaybackStatus.StartedAt == "" &&
		(CurrentPlaybackItem.Category == "music" || CurrentPlaybackItem.Category == "podcast") {
		return // Spotify is/was last active + trakt is not so no point continuing
	}

	var category string

	if traktResponse.MediaType == "movie" {
		category = traktResponse.MediaType
	}

	if traktResponse.MediaType == "episode" {
		category = "tv"
	}

	playingItem := models.MediaItem{
		IsActive:  true,
		Category:  category,
	}

	var playbackProgress models.MediaProgress

	// 2022-01-01T22:53:59.000Z
	startedAt, err := time.Parse("2006-01-02T15:04:05.000Z", MediaPlaybackStatus.StartedAt)
	if err != nil {
		log.Printf("Failed to parse trakt timestamp %s", MediaPlaybackStatus.StartedAt)
	}
	playingItem.StartedAt = float64(startedAt.UnixMilli())
	playbackProgress.StartedAt = float64(startedAt.UnixMilli())

	var (
		backdrops models.Backdrops
		posters   models.Posters
		stills    models.Stills
	)

	if traktResponse.MediaType == "episode" {
		showURL := fmt.Sprintf(
			ShowImageEndpoint,
			traktResponse.Show.IDs.TMDB,
		)
		showImageResponse := getMediaImage(showURL)
		err := json.Unmarshal(showImageResponse, &backdrops)
		if err != nil {
			fmt.Println("Error fetching show images from TMDB: ", err)
		}
		MediaPlaybackStatus.Show.Backdrops = buildAbsoluteImageLink(backdrops.Backdrops)

		seasonURL := fmt.Sprintf(
			SeasonImageEndpoint,
			traktResponse.Show.IDs.TMDB,
			traktResponse.Episode.SeasonNumber,
		)
		seasonImageResponse := getMediaImage(seasonURL)
		err = json.Unmarshal(seasonImageResponse, &posters)
		if err != nil {
			fmt.Println("Error fetching season images from TMDB: ", err)
		}
		MediaPlaybackStatus.Episode.SeasonPosters = buildAbsoluteImageLink(posters.Posters)

		episodeURL := fmt.Sprintf(
			EpisodeImageEndpoint,
			traktResponse.Show.IDs.TMDB,
			traktResponse.Episode.SeasonNumber,
			traktResponse.Episode.EpisodeNumber,
		)
		episodeImageResponse := getMediaImage(episodeURL)
		err = json.Unmarshal(episodeImageResponse, &stills)
		if err != nil {
			fmt.Println("Error fetching episode images from TMDB: ", err)
		}
		MediaPlaybackStatus.Episode.EpisodeStills = buildAbsoluteImageLink(stills.Stills)

		mergedTitle := fmt.Sprintf(
			"%02dx%02d %s",
			traktResponse.Episode.SeasonNumber,
			traktResponse.Episode.EpisodeNumber,
			traktResponse.Episode.Title,
		)

		playingItem.Title = mergedTitle
		playingItem.TitleLink = MediaPlaybackStatus.Episode.Link
		playingItem.Subtitle = MediaPlaybackStatus.Show.Title
		playingItem.SubtitleLink = MediaPlaybackStatus.Show.Link
		playingItem.Duration = MediaPlaybackStatus.Episode.Runtime * 60000
		playingItem.PreviewURL = MediaPlaybackStatus.Show.Trailer

		playbackProgress.Duration = MediaPlaybackStatus.Episode.Runtime * 60000

		var showImages []models.MediaImage
		showImages = append(showImages, models.MediaImage{
			URL:    MediaPlaybackStatus.Show.Backdrops[0].FilePath,
			Height: MediaPlaybackStatus.Show.Backdrops[0].Height,
			Width:  MediaPlaybackStatus.Show.Backdrops[0].Width,
		})
		playingItem.Images = showImages
	}

	if traktResponse.MediaType == "movie" {
		movieURL := fmt.Sprintf(
			MovieImageEndpoint,
			traktResponse.Movie.IDs.TMDB,
		)
		movieImageResponse := getMediaImage(movieURL)
		err = json.Unmarshal(movieImageResponse, &posters)
		if err != nil {
			fmt.Println("Error fetching movie images from TMDB: ", err)
		}
		MediaPlaybackStatus.Movie.Poster = buildAbsoluteImageLink(posters.Posters)

		playingItem.Title = MediaPlaybackStatus.Movie.Title
		playingItem.TitleLink = MediaPlaybackStatus.Movie.Link
		playingItem.Subtitle = fmt.Sprintf("%d | %s", MediaPlaybackStatus.Movie.Year, MediaPlaybackStatus.Movie.Certification)
		playingItem.Duration = MediaPlaybackStatus.Movie.Runtime * 60000
		playingItem.PreviewURL = MediaPlaybackStatus.Movie.Trailer

		var posterImages []models.MediaImage
		posterImages = append(posterImages, models.MediaImage{
			URL:    MediaPlaybackStatus.Movie.Poster[0].FilePath,
			Height: MediaPlaybackStatus.Movie.Poster[0].Height,
			Width:  MediaPlaybackStatus.Movie.Poster[0].Width,
		})
		playingItem.Images = posterImages
	}

	playbackProgress.Duration = MediaPlaybackStatus.Movie.Runtime * 60000

	CurrentPlaybackItem = playingItem

	return
}
