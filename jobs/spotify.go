package jobs

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gregdel/pushover"
	"log"
	"os"

	"github.com/marcus-crane/gunslinger/models"
)

var (
	currentToken        models.Token
	AudioPlaybackStatus models.Audio
)

const (
	SlackStatusEndpoint = "https://slack.com/api/users.profile.set"
	RefreshEndpoint     = "https://accounts.spotify.com/api/token"
	PlayerEndpoint      = "https://api.spotify.com/v1/me/player/currently-playing?market=NZ&additional_types=episode"
	UserAgent           = "Now Playing/1.0 (utf9k.net)"
)

func RefreshAccessToken() {
	refreshToken := os.Getenv("SPOTIFY_REFRESH_TOKEN")
	refreshAuthHeader := os.Getenv("SPOTIFY_REFRESH_BASIC_AUTH")

	authHeader := fmt.Sprintf("Basic %s", refreshAuthHeader)

	args := fiber.AcquireArgs()

	args.Set("grant_type", "refresh_token")
	args.Set("refresh_token", refreshToken)

	tokenA := fiber.Post(RefreshEndpoint).
		UserAgent(UserAgent).
		Form(args).
		Add("Authorization", authHeader)

	var tokenResponse models.Token

	_, body, errs := tokenA.Bytes() // TODO: Check response code is what we hope for

	if len(errs) != 0 {
		panic(errs)
	}

	err := json.Unmarshal(body, &tokenResponse)

	if err != nil {
		fmt.Println("error: ", err)
	}

	currentToken = tokenResponse

	fiber.ReleaseArgs(args)
}

func GetCurrentlyPlaying() {

	slackToken := os.Getenv("SLACK_TOKEN")

	if currentToken.AccessToken == "" {
		fmt.Println("No access token retrieved yet. Skipping out on getting currently playing songs.")
		return
	}

	authHeader := fmt.Sprintf("Bearer %s", currentToken.AccessToken)

	playerA := fiber.Get(PlayerEndpoint).
		UserAgent(UserAgent).
		Add("Authorization", authHeader)

	var playerResponse models.Audio

	code, body, errs := playerA.Bytes()

	if len(errs) != 0 {
		panic(errs)
	}

	if code == 429 {
		fmt.Println("Rate limited! Sending a pushover notification.")
		app := pushover.New(os.Getenv("PUSHOVER_APP_TOKEN"))
		recipient := pushover.NewRecipient(os.Getenv("PUSHOVER_USER_ID"))
		message := &pushover.Message{
			Message:    fmt.Sprintf("A 429 error code was detected when trying to request the currently playing song."),
			Title:      "Gunslinger was rate limited by Spotify",
			URL:        "https://developer.spotify.com/documentation/web-api/",
			URLTitle:   "Spotify Web API documentation",
			DeviceName: "iPhone12Pro",
		}
		_, err := app.SendMessage(message, recipient)
		if err != nil {
			// Just continue since the next block will handle things for us anyway
			fmt.Println(err)
		}
	}

	if code != 200 {
		return // A song isn't currently playing
	}

	err := json.Unmarshal(body, &playerResponse)

	if err != nil {
		fmt.Println("error: ", err)
	}

	progress := float64(playerResponse.Progress)
	duration := float64(playerResponse.Item.Duration)

	playerResponse.PercentDone = progress / duration * 100

	AudioPlaybackStatus = playerResponse

	if !AudioPlaybackStatus.CurrentlyPlaying && CurrentPlaybackItem.IsActive {
		// We are transitioning into stopped music state so stub out Slack status
		status := fiber.Map{
			"profile": fiber.Map{
				"status_text":  "",
				"status_emoji": "",
			},
		}
		slackA := fiber.Post(SlackStatusEndpoint).
			JSON(status).
			UserAgent(UserAgent).
			Add("Authorization", fmt.Sprintf("Bearer %s", slackToken)).
			Add("Content-Type", "application/json; charset=utf-8")

		_, body, errs := slackA.Bytes()
		log.Print(errs)
		log.Print(string(body))
	}

	if !AudioPlaybackStatus.CurrentlyPlaying &&
		(CurrentPlaybackItem.Category == "music" || CurrentPlaybackItem.Category == "podcast") {
		CurrentPlaybackItem.IsActive = false
		CurrentPlaybackProgress.IsActive = false
		return // Nothing playing but we want to retain the last played track
	}

	if !AudioPlaybackStatus.CurrentlyPlaying &&
		(CurrentPlaybackItem.Category == "tv" || CurrentPlaybackItem.Category == "movie") {
		return // Plex is/was last active + trakt is not so no point continuing
	}

	var category string

	if AudioPlaybackStatus.AudioType == "episode" {
		category = "podcast"
	}

	if AudioPlaybackStatus.AudioType == "track" {
		category = "music"
	}

	playingItem := models.MediaItem{
		StartedAt:       playerResponse.Timestamp,
		IsActive:        playerResponse.CurrentlyPlaying,
		PercentComplete: playerResponse.PercentDone,
		Elapsed:         playerResponse.Progress,
		Duration:        playerResponse.Item.Duration,
		PreviewURL:      playerResponse.Item.PreviewURL,
		Title:           playerResponse.Item.Name,
		TitleLink:       playerResponse.Item.Link.SpotifyURL,
		Category:        category,
	}

	CurrentPlaybackProgress = models.MediaProgress{
		StartedAt:       playerResponse.Timestamp,
		IsActive:        playerResponse.CurrentlyPlaying,
		PercentComplete: playerResponse.PercentDone,
		Elapsed:         playerResponse.Progress,
		Duration:        playerResponse.Item.Duration,
	}

	if playerResponse.AudioType == "track" {
		playingItem.Subtitle = playerResponse.Item.Album.Artists[0].Name
		playingItem.SubtitleLink = playerResponse.Item.Album.Artists[0].Link.SpotifyURL

		var trackImages []models.MediaImage
		for _, entry := range playerResponse.Item.Album.Images {
			trackImages = append(trackImages, models.MediaImage{
				URL:    entry.URL,
				Height: entry.Height,
				Width:  entry.Width,
			})
		}
		playingItem.Images = trackImages
	}

	if playerResponse.AudioType == "episode" {
		playingItem.Subtitle = playerResponse.Item.Podcast.Name
		playingItem.SubtitleLink = playerResponse.Item.Podcast.Link.SpotifyURL

		var podcastImages []models.MediaImage
		for _, entry := range playerResponse.Item.Podcast.Images {
			podcastImages = append(podcastImages, models.MediaImage{
				URL:    entry.URL,
				Height: entry.Height,
				Width:  entry.Width,
			})
		}
		playingItem.Images = podcastImages
	}

	if playingItem.Title != CurrentPlaybackItem.Title || CurrentPlaybackItem.IsActive == false {
		status := fiber.Map{
			"profile": fiber.Map{
				"status_text":  fmt.Sprintf("Listening to %s by %s", playingItem.Title, playingItem.Subtitle),
				"status_emoji": ":spotify-new:",
			},
		}
		_ = fiber.Post(SlackStatusEndpoint).
			JSON(status).
			UserAgent(UserAgent).
			Add("Authorization", fmt.Sprintf("Bearer %s", slackToken)).
			Add("Content-Type", "application/json; charset=utf-8")

		//_, body, errs := slackA.Bytes()
		//log.Print(errs)
		//log.Print(string(body))
	}

	CurrentPlaybackItem = playingItem
}
