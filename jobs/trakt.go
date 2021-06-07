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
	TraktWatchingEndpoint  = "https://api.trakt.tv/users/sentry/watching"
	userAgent              = "Now Playing/1.0 (utf9k.net)"
)

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
}
