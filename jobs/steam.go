package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"

	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/r3labs/sse/v2"
	"gorm.io/gorm"
)

var (
	profileEndpoint    = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=76561197999386785"
	gameDetailEndpoint = "https://store.steampowered.com/api/appdetails?appids=%s"
)

func GetCurrentlyPlayingSteam(database *gorm.DB) {

	steamApiKey := os.Getenv("STEAM_TOKEN")
	playingUrl := fmt.Sprintf(profileEndpoint, steamApiKey)

	var client http.Client
	req, err := http.NewRequest("GET", playingUrl, nil)
	if err != nil {
		panic(err)
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var steamResponse models.SteamPlayerSummary

	if err = json.Unmarshal(body, &steamResponse); err != nil {
		fmt.Println("Error fetching Steam data: ", err)
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
		panic(err)
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{UserAgent},
	}
	res, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var gameDetailResponse map[string]models.SteamAppResponse

	if err = json.Unmarshal(body, &gameDetailResponse); err != nil {
		fmt.Println("Error fetching Steam app data: ", err)
	}

	game := gameDetailResponse[gameId].Data

	developer := "Unknown Developer"

	if len(game.Developers) > 0 {
		developer = game.Developers[0]
	}

	_, dominantColours := extractImageContent(game.HeaderImage)

	playingItem := models.MediaItem{
		Title:           game.Name,
		Subtitle:        developer,
		Category:        "gaming",
		Source:          "steam",
		Image:           game.HeaderImage,
		IsActive:        true,
		DominantColours: dominantColours,
	}

	// reflect.DeepEqual is good enough for our purposes even though
	// it doesn't do things like properly copmare timestamp metadata.
	// For just checking if we should emit a message, it's good enough
	if !reflect.DeepEqual(CurrentPlaybackItem, playingItem) {
		byteStream := new(bytes.Buffer)
		json.NewEncoder(byteStream).Encode(playingItem)
		events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		// We want to make sure that we don't resave if the server restarts
		// to ensure the history endpoint is relatively accurate
		var previousItem models.DBMediaItem
		database.Where("category = ?", playingItem.Category).Last(&previousItem)
		if CurrentPlaybackItem.Title != playingItem.Title && previousItem.Title != playingItem.Title {
			dbItem := models.DBMediaItem{
				Title:    playingItem.Title,
				Subtitle: playingItem.Subtitle,
				Category: playingItem.Category,
				IsActive: playingItem.IsActive,
				Source:   playingItem.Source,
				Image:    playingItem.Image,
			}
			database.Save(&dbItem)
		}
	}

	CurrentPlaybackItem = playingItem
}
