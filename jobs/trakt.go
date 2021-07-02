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
  EpisodeImageEndpoint  = "https://api.themoviedb.org/3/tv/%d/season/%d/episode/%d/images?api_key=%s"
  SeasonImageEndpoint   = "https://api.themoviedb.org/3/tv/%d/season/%d/images?api_key=%s"
  ShowImageEndpoint     = "https://api.themoviedb.org/3/tv/%d/images?api_key=%s"
  MovieImageEndpoint    = "https://api.themoviedb.org/3/movie/%d/images?api_key=%s"
	userAgent             = "Now Playing/1.0 (utf9k.net)"
)

func getMediaImage(imageURL string) models.Image {
  imageA := fiber.Get(
      fmt.Sprintf(imageURL),
    ).
    UserAgent(userAgent)

  var imageResponse models.Image

  code, body, errs := imageA.Bytes()

  if len(errs) != 0 {
    panic(errs)
  }

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

  tmdbApiKey := os.Getenv("TMDB_API_KEY")

  if traktResponse.MediaType == "episode" {
    showURL := fmt.Sprintf(
      ShowImageEndpoint,
      traktResponse.Show.IDs.TMDB,
      tmdbApiKey,
    )
    showImage := getMediaImage(showURL)
    MediaPlaybackStatus.Show.Poster = showImage

    seasonURL := fmt.Sprintf(
      SeasonImageEndpoint,
      traktResponse.Show.IDs.TMDB,
      traktResponse.Episode.SeasonNumber,
      tmdbApiKey,
    )
    seasonImage := getMediaImage(seasonURL)
    MediaPlaybackStatus.Episode.SeasonPoster = seasonImage

    episodeURL := fmt.Sprintf(
      EpisodeImageEndpoint,
      traktResponse.Show.IDs.TMDB,
      traktResponse.Episode.SeasonNumber,
      traktResponse.Episode.EpisodeNumber,
      tmdbApiKey,
    )
    episodeImage := getMediaImage(episodeURL)
    MediaPlaybackStatus.Episode.EpisodeStill = episodeImage

  }

  if traktResponse.MediaType == "movie" {
    movieURL := fmt.Sprintf(
      MovieImageEndpoint,
      traktResponse.Movie.IDs.TMDB,
      tmdbApiKey,
    )
    movieImages := getMediaImage(movieURL)
    MediaPlaybackStatus.Movie.Images = movieImages
  }

}
