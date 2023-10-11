package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/shared"
)

const (
	ERR_STORE_RESPONSE_MALFORMED = "failed to find game id in store request"
)

type Client struct {
	APIKey       string
	APIBaseURL   string
	StoreBaseURL string
	HTTPClient   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:       apiKey,
		APIBaseURL:   "https://api.steampowered.com",
		StoreBaseURL: "https://store.steampowered.com/api",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) getUserPlaying() (string, error) {
	var steamResponse SteamPlayerSummary
	var activeTitle string
	endpoint := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=76561197999386785", c.APIBaseURL, c.APIKey)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return activeTitle, err
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{"Gunslinger/2.0 <github.com/marcus-crane/gunslinger>"},
	}
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return activeTitle, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return activeTitle, err
	}
	if err := json.Unmarshal(body, &steamResponse); err != nil {
		return activeTitle, err
	}
	if len(steamResponse.Response.Players) == 0 {
		return activeTitle, err
	}
	// There is only ever one active player but we'll just use
	// a range to handle the unlikely case of no players found
	for _, player := range steamResponse.Response.Players {
		if player.GameID != "" {
			activeTitle = player.GameID
		}
	}
	return activeTitle, nil
}

func (c *Client) lookupStoreItem(appID string) (SteamAppDetail, error) {
	var steamResponse map[string]SteamAppResponse
	endpoint := fmt.Sprintf("%s/appdetails?appids=%s", c.StoreBaseURL, appID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return SteamAppDetail{}, err
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{"Gunslinger/2.0 <github.com/marcus-crane/gunslinger>"},
	}
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return SteamAppDetail{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return SteamAppDetail{}, err
	}
	if err := json.Unmarshal(body, &steamResponse); err != nil {
		return SteamAppDetail{}, err
	}
	game, ok := steamResponse[appID]
	if !ok {
		return SteamAppDetail{}, fmt.Errorf(ERR_STORE_RESPONSE_MALFORMED)
	}
	return game.Data, nil
}

func (c *Client) QueryMediaState() (shared.DBMediaItem, error) {
	userPlayingID, err := c.getUserPlaying()
	if err != nil {
		return shared.DBMediaItem{}, err
	}
	if userPlayingID == "" {
		return shared.DBMediaItem{}, fmt.Errorf("empty game id found")
	}
	storeDetail, err := c.lookupStoreItem(userPlayingID)
	if err != nil {
		return shared.DBMediaItem{}, err
	}
	return shared.DBMediaItem{
		Title:    storeDetail.Name,
		Author:   storeDetail.Developers[0],
		Category: "gaming",
		IsActive: true,
		Source:   "steam",
	}, nil
}

type SteamPlayerSummary struct {
	Response SteamResponse `json:"response"`
}

type SteamResponse struct {
	Players []SteamUser `json:"players"`
}

type SteamUser struct {
	GameID                   string `json:"gameid"`
	CommunityVisibilityState int    `json:"communityvisibilitystate"`
	ProfileState             int    `json:"profilestate"`
	PersonaState             int    `json:"personastate"`
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
