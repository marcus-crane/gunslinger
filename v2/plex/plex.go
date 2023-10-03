package plex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/v2/shared"
)

const (
	SESSION_ENDPOINT = "/status/sessions"
)

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: "http://netocean:32400",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// TODO: Use a URL builder to ensure slashes are normalised
func (c *Client) buildUrl(endpoint string) string {
	return fmt.Sprintf("%s%s?X-Plex-Token=%s", c.BaseURL, endpoint, c.APIKey)
}

func (c *Client) getUserPlaying() (PlexResponse, error) {
	var plexResponse PlexResponse
	endpoint := c.buildUrl(SESSION_ENDPOINT)
	fmt.Println(endpoint)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return plexResponse, err
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{"Gunslinger/2.0 <github.com/marcus-crane/gunslinger>"},
	}
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return plexResponse, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return plexResponse, err
	}
	if err := json.Unmarshal(body, &plexResponse); err != nil {
		return plexResponse, err
	}
	return plexResponse, err
}

func (c *Client) QueryMediaState() ([]shared.DBMediaItem, error) {
	var plexState []shared.DBMediaItem
	status, err := c.getUserPlaying()
	if err != nil {
		return plexState, err
	}
	if status.MediaContainer.Size == 0 {
		return plexState, err
	}
	for _, entry := range status.MediaContainer.Metadata {
		// We don't want to capture movie trailers as historical items
		if entry.Type == "clip" {
			continue
		}
		mediaItem := shared.DBMediaItem{
			Title:    entry.Title,
			Subtitle: entry.GrandparentTitle,
			Category: entry.Type,
			Elapsed:  entry.ViewOffset,
			Duration: entry.Duration,
			Source:   "plex",
		}
		if entry.Player.State == "playing" {
			mediaItem.IsActive = true
		}
		if entry.Type == "episode" {
			mediaItem.Title = fmt.Sprintf(
				"%02dx%02d %s",
				entry.ParentIndex, // Season number
				entry.Index,       // Episode number
				entry.Title,
			)
		}
		if entry.Type == "movie" {
			mediaItem.Author = entry.Director[0].Name
		}

		if entry.Type == "track" {
			mediaItem.Subtitle = entry.ParentTitle
			mediaItem.Author = entry.GrandparentTitle
		}

		plexState = append(plexState, mediaItem)
	}
	return plexState, nil
}

type PlexResponse struct {
	MediaContainer MediaContainer `json:"MediaContainer"`
}

type MediaContainer struct {
	Size     int        `json:"size"`
	Metadata []Metadata `json:"Metadata"`
}

type Metadata struct {
	Attribution      string     `json:"attribution"`
	Duration         int        `json:"duration"`
	GrandparentTitle string     `json:"grandparentTitle"`
	Thumb            string     `json:"thumb"`
	ParentThumb      string     `json:"parentThumb"`
	Index            int        `json:"index"`
	ParentIndex      int        `json:"parentIndex"`
	ParentTitle      string     `json:"parentTitle"`
	Title            string     `json:"title"`
	Type             string     `json:"type"`
	ViewOffset       int        `json:"viewOffset"`
	Director         []Director `json:"Director"`
	Player           Player     `json:"Player"`
}

type Director struct {
	Name string `json:"tag"`
}

type Player struct {
	State string `json:"state"`
}
