package models

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type MediaItem struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Category string `json:"category"`
	IsActive bool   `json:"is_active"`
	Elapsed  int    `json:"elapsed_ms"`
	Duration int    `json:"duration_ms"`
	Image    string `json:"image"`
}

type SlackProfile struct {
	StatusText       string  `json:"status_text"`
	StatusEmoji      string  `json:"status_emoji"`
	StatusExpiration float64 `json:"status_expiration"`
}
