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

type Album struct {
	Artists []Artist `json:"artists"`
}

type AudioItem struct {
	Name       string       `json:"name"`
	Link       ExternalURLs `json:"external_urls"`
	Podcast    Podcast      `json:"show"`
	Album      Album        `json:"album"`
	PreviewURL string       `json:"preview_url"`
	Duration   int          `json:"duration_ms"`
}

type ExternalURLs struct {
	SpotifyURL string `json:"spotify"`
}
