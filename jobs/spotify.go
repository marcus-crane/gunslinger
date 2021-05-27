package jobs

import (
  "fmt"
  "encoding/json"
  "os"

  "github.com/gofiber/fiber/v2"
)

var currentToken Token

const (
  RefreshEndpoint = "https://accounts.spotify.com/api/token"
  PlayerEndpoint  = "https://api.spotify.com/v1/me/player/currently-playing?market=NZ&additional_types=episode"
  UserAgent       = "Now Playing/1.0 (utf9k.net)"
)

type Token struct {
  AccessToken string `json:"access_token"`
  TokenType   string `json:"token_type"`
  ExpiresIn   int    `json:"expires_in"`
  Scope       string `json:"scope"`
}

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

  var tokenResponse Token

  _, body, errs := tokenA.Bytes() // TODO: Check response code is what we hope for

  if len(errs) != 0 {
    panic(errs)
  }

  err := json.Unmarshal(body, &tokenResponse)

  if err != nil {
    fmt.Println("error: ", err)
  }

  currentToken = tokenResponse

  fmt.Println(currentToken.ExpiresIn)

  fiber.ReleaseArgs(args)
}
