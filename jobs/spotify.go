package jobs

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/models"
)

var (
	currentToken        models.Token
	AudioPlaybackStatus models.Audio
)

const (
	RefreshEndpoint = "https://accounts.spotify.com/api/token"
	PlayerEndpoint  = "https://api.spotify.com/v1/me/player/currently-playing?market=NZ&additional_types=episode"
	UserAgent       = "Now Playing/1.0 (utf9k.net)"
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

	fmt.Println(code)

	if len(errs) != 0 {
		panic(errs)
	}

	if code != 200 {
		fmt.Println("Non-200 code from Spotify. No audio playing most likely.")
		AudioPlaybackStatus = models.Audio{}
		return // A song isn't currently playing
	}

	err := json.Unmarshal(body, &playerResponse)

	if err != nil {
		fmt.Println("error: ", err)
	}

	progress := float64(playerResponse.Progress)
	duration := float64(playerResponse.Item.Duration)

	playerResponse.PercentDone = (progress / duration * 100)

	AudioPlaybackStatus = playerResponse

	fmt.Println("Updated Spotify playback status")
}
