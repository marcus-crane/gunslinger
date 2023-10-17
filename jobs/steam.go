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
	profileEndpoint    = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=76561197999386785"
	gameDetailEndpoint = "https://store.steampowered.com/api/appdetails?appids=%s"
)

func GetCurrentlyPlayingSteam(store db.Store, client http.Client) {

	steamApiKey := utils.MustEnv("STEAM_TOKEN")
	playingUrl := fmt.Sprintf(profileEndpoint, steamApiKey)

	req, err := http.NewRequest("GET", playingUrl, nil)
	if err != nil {
		slog.Error("Failed to prepare Steam request",
			slog.String("stack", err.Error()),
		)
		return
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to contact Steam for updates",
			slog.String("stack", err.Error()),
		)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read Steam response",
			slog.String("stack", err.Error()),
		)
		return
	}
	var steamResponse models.SteamPlayerSummary

	if err = json.Unmarshal(body, &steamResponse); err != nil {
		slog.Error("Error fetching Steam data",
			slog.String("stack", err.Error()),
		)
	}

	if len(steamResponse.Response.Players) == 0 {
		if CurrentPlaybackItem.Source == "steam" {
			CurrentPlaybackItem.IsActive = false
		}
		return
	}

	gameId := steamResponse.Response.Players[0].GameID

	if gameId == "" {
		if CurrentPlaybackItem.Source == "steam" {
			CurrentPlaybackItem.IsActive = false
		}
		return
	}

	gameDetailUrl := fmt.Sprintf(gameDetailEndpoint, gameId)

	req, err = http.NewRequest("GET", gameDetailUrl, nil)
	if err != nil {
		slog.Error("Failed to prepare Steam request for more detail",
			slog.String("stack", err.Error()),
		)
		return
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{utils.UserAgent},
	}
	res, err = client.Do(req)
	if err != nil {
		slog.Error("Failed to read Steam detail response",
			slog.String("stack", err.Error()),
		)
		return
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read Steam detail response",
			slog.String("stack", err.Error()),
		)
		return
	}
	var gameDetailResponse map[string]models.SteamAppResponse

	if err = json.Unmarshal(body, &gameDetailResponse); err != nil {
		slog.Error("Error fetching Steam app data",
			slog.String("stack", err.Error()),
		)
	}

	game := gameDetailResponse[gameId].Data

	developer := "Unknown Developer"

	if len(game.Developers) > 0 {
		developer = game.Developers[0]
	}

	image, extension, dominantColours, err := utils.ExtractImageContent(game.HeaderImage)
	if err != nil {
		slog.Error("Failed to extract image content",
			slog.String("stack", err.Error()),
			slog.String("image_url", game.HeaderImage),
		)
		return
	}

	imageLocation, guid := utils.BytesToGUIDLocation(image, extension)

	playingItem := models.MediaItem{
		CreatedAt:       time.Now().Unix(),
		Title:           game.Name,
		Subtitle:        developer,
		Category:        "gaming",
		Source:          "steam",
		IsActive:        true,
		Duration:        0,
		DominantColours: dominantColours,
		Image:           imageLocation,
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
					slog.Error("Failed to save cover for Steam",
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
