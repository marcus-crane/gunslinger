package models

import "time"

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type DBMediaItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	Category  string    `json:"category"`
	IsActive  bool      `json:"is_active"`
	Source    string    `json:"source"`
}

type ResponseMediaItem struct {
	OccuredAt       string   `json:"occurred_at"`
	Title           string   `json:"title"`
	Subtitle        string   `json:"subtitle"`
	Category        string   `json:"category"`
	Source          string   `json:"source"`
	DominantColours []string `json:"dominant_colours"`
}

type MediaItem struct {
	Title           string   `json:"title"`
	Subtitle        string   `json:"subtitle"`
	Category        string   `json:"category"`
	IsActive        bool     `json:"is_active"`
	Elapsed         int      `json:"elapsed_ms"`
	Duration        int      `json:"duration_ms"`
	Source          string   `json:"source"`
	Image           string   `json:"image"`
	DominantColours []string `json:"dominant_colours"`
	Backfilled      bool     `json:"-"`
}

type SlackProfile struct {
	StatusText       string  `json:"status_text"`
	StatusEmoji      string  `json:"status_emoji"`
	StatusExpiration float64 `json:"status_expiration"`
}
