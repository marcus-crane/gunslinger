package models

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type MediaItem struct {
	Title           string       `json:"title"`
	TitleLink       string       `json:"title_link"`
	Subtitle        string       `json:"subtitle"`
	SubtitleLink    string       `json:"subtitle_link"`
	Category        string       `json:"category"`
	StartedAt       float64      `json:"started_at"`
	IsActive        bool         `json:"is_active"`
	Elapsed         int          `json:"elapsed_ms"`
	Duration        int          `json:"duration_ms"`
	PercentComplete float64      `json:"percent_complete"`
	PreviewURL      string       `json:"preview_url"`
	Images          []MediaImage `json:"images"`
}

type MediaImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type MediaProgress struct {
	StartedAt       float64 `json:"started_at"`
	IsActive        bool    `json:"is_active"`
	Elapsed         int     `json:"elapsed_ms"`
	Duration        int     `json:"duration_ms"`
	PercentComplete float64 `json:"percent_complete"`
}

type SlackProfile struct {
	StatusText       string  `json:"status_text"`
	StatusEmoji      string  `json:"status_emoji"`
	StatusExpiration float64 `json:"status_expiration"`
}
