package jobs

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/models"
)

var (
	MediaPlaybackStatus models.Media
)

const (
	TraktWatchingEndpoint = "https://api.trakt.tv/users/sentry/watching"
	EpisodeImageEndpoint  = "https://api.themoviedb.org/3/tv/%d/season/%d/episode/%d/images"
	SeasonImageEndpoint   = "https://api.themoviedb.org/3/tv/%d/season/%d/images"
	ShowImageEndpoint     = "https://api.themoviedb.org/3/tv/%d/images"
	MovieImageEndpoint    = "https://api.themoviedb.org/3/movie/%d/images"
	userAgent             = "Now Playing/1.0 (utf9k.net)"
)

func getMediaImage(imageURL string) models.Image {
	tmdbApiKey := os.Getenv("TMDB_API_KEY")

	imageA := fiber.Get(imageURL).
		UserAgent(userAgent).
		Add("Content-Type", "application/json;charset=utf-8").
		Add("Authorzation", fmt.Sprintf("Bearer %s", tmdbApiKey))

	var imageResponse models.Image

	code, body, errs := imageA.Bytes()

	if len(errs) != 0 {
		panic(errs)
	}

	fmt.Printf("Fetch image %s", imageURL)
	fmt.Println(code)

	err := json.Unmarshal(body, &imageResponse)

	if err != nil {
		fmt.Println("error: ", err)
	}

	return imageResponse
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

	fmt.Println(code)

	err := json.Unmarshal(body, &traktResponse)

	if err != nil {
		fmt.Println("error: ", err)
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

	MediaPlaybackStatus = traktResponse

	fmt.Println("Updated media playback status")

	if traktResponse.MediaType == "episode" {
		fmt.Println("Updating episode cover art")
		showURL := fmt.Sprintf(
			ShowImageEndpoint,
			traktResponse.Show.IDs.TMDB,
		)
		fmt.Println(showURL)
		fmt.Println("Show image url")
		showImage := getMediaImage(showURL)
		MediaPlaybackStatus.Show.Poster = showImage

		seasonURL := fmt.Sprintf(
			SeasonImageEndpoint,
			traktResponse.Show.IDs.TMDB,
			traktResponse.Episode.SeasonNumber,
		)
		seasonImage := getMediaImage(seasonURL)
		fmt.Println(seasonImage)
		MediaPlaybackStatus.Episode.SeasonPoster = seasonImage

		episodeURL := fmt.Sprintf(
			EpisodeImageEndpoint,
			traktResponse.Show.IDs.TMDB,
			traktResponse.Episode.SeasonNumber,
			traktResponse.Episode.EpisodeNumber,
		)
		episodeImage := getMediaImage(episodeURL)
		MediaPlaybackStatus.Episode.EpisodeStill = episodeImage
		return
	}

	if traktResponse.MediaType == "movie" {
		movieURL := fmt.Sprintf(
			MovieImageEndpoint,
			traktResponse.Movie.IDs.TMDB,
		)
		movieImages := getMediaImage(movieURL)
		MediaPlaybackStatus.Movie.Poster = movieImages
		return
	}

}
