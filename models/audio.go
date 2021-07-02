package models

type Audio struct {
	AudioType        string    `json:"currently_playing_type"`
	Timestamp        float64   `json:"timestamp"`
	CurrentlyPlaying bool      `json:"is_playing"`
	Progress         int       `json:"progress_ms"`
	PercentDone      float64   `json:"percent_done"`
	Item             AudioItem `json:"item"`
}

type Podcast struct {
	Name string       `json:"name"`
	Link ExternalURLs `json:"external_urls"`
}

type Artist struct {
	Name string       `json:"name"`
	Link ExternalURLs `json:"external_urls"`
}

type Cover struct {
	Height int    `json:"height"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
}

type Album struct {
	Artists []Artist     `json:"artists"`
	Images  []Cover      `json:"images"`
	Name    string       `json:"name"`
	Link    ExternalURLs `json:"external_urls"`
}

type AudioItem struct {
	Name       string       `json:"name"`
	Link       ExternalURLs `json:"external_urls"`
	Podcast    Podcast      `json:"show"`
	Album      Album        `json:"album"`
	PreviewURL string       `json:"preview_url"`
	Duration   int          `json:"duration_ms"`
	Explicit   bool         `json:"explicit"`
}

type ExternalURLs struct {
	SpotifyURL string `json:"spotify"`
}
