package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/utils"
)

var (
	profileEndpoint    = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=76561197999386785"
	gameDetailEndpoint = "https://store.steampowered.com/api/appdetails?appids=%s"
)

type SteamPlayerSummary struct {
	Response SteamResponse `json:"response"`
}

type SteamResponse struct {
	Players []SteamUser `json:"players"`
}

type SteamUser struct {
	GameID       string `json:"gameid"`
	PersonaState int    `json:"personastate"`
}

type SteamAppResponse struct {
	Data SteamAppDetail `json:"data"`
}

type SteamAppDetail struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	HeaderImage string   `json:"header_image"`
	Developers  []string `json:"developers"`
}

func GetCurrentlyPlaying(cfg config.Config, ps *playback.PlaybackSystem, client http.Client) {
	playingUrl := fmt.Sprintf(profileEndpoint, cfg.Steam.Token)

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
	var steamResponse SteamPlayerSummary

	if err = json.Unmarshal(body, &steamResponse); err != nil {
		slog.Error("Error fetching Steam data",
			slog.String("stack", err.Error()),
		)
		return
	}

	if len(steamResponse.Response.Players) == 0 {
		ps.DeactivateBySource(string(playback.Steam))
		return
	}

	gameId := steamResponse.Response.Players[0].GameID

	if gameId == "" {
		ps.DeactivateBySource(string(playback.Steam))
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
	var gameDetailResponse map[string]SteamAppResponse

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

	image, extension, domColours, err := utils.ExtractImageContent(game.HeaderImage)
	if err != nil {
		slog.Error("Failed to extract image content",
			slog.String("stack", err.Error()),
			slog.String("image_url", game.HeaderImage),
		)
		return
	}

	imageLocation, _ := utils.BytesToGUIDLocation(image, extension)

	update := playback.Update{
		MediaItem: playback.MediaItem{
			Title:           game.Name,
			Subtitle:        developer,
			Category:        string(playback.Gaming),
			Duration:        0,
			Source:          string(playback.Steam),
			Image:           imageLocation,
			DominantColours: domColours,
		},
		Status: playback.StatusPlaying,
	}

	if err := ps.UpdatePlaybackState(update); err != nil {
		slog.Error("Failed to save Steam update",
			slog.String("stack", err.Error()),
			slog.String("title", update.MediaItem.Title))
	}

	hash := playback.GenerateMediaID(&update)
	if err := utils.SaveCover(cfg, hash, image, extension); err != nil {
		slog.Error("Failed to save cover for Steam",
			slog.String("stack", err.Error()),
			slog.String("guid", hash),
			slog.String("title", update.MediaItem.Title),
		)
	}
}
