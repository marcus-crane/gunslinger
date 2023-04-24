package models

import (
	"database/sql/driver"
	"errors"
	"strings"
)

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type DBMediaItem struct {
	ID              uint   `gorm:"primaryKey" json:"id" db:"id"`
	CreatedAt       int64  `json:"created_at" db:"created_at"`
	Title           string `json:"title" db:"title"`
	Subtitle        string `json:"subtitle" db:"subtitle"`
	Category        string `json:"category" db:"category"`
	IsActive        bool   `json:"is_active" db:"is_active"`
	DurationMs      int    `json:"duration_ms" db:"duration_ms"`
	DominantColours string `json:"dominant_colours" db:"dominant_colours"`
	Source          string `json:"source" db:"source"`
	Image           string `json:"image" db:"image"`
}

// SerializedColours is a custom DB extension type that stores
// a string slice as a comma separate value in the database
// Example input: []string{"#020304", "#6581be"}
// Example DB value: #020304,#6581be
type SerializedColors []string

func (s SerializedColors) Value() (driver.Value, error) {
	return strings.Join(s, ","), nil
}

func (s *SerializedColors) Scan(src interface{}) error {
	var source []string
	switch src.(type) {
	case string:
		source = strings.Split(src.(string), ",")
	default:
		return errors.New("incompatible type for SerializedColors")
	}
	*s = SerializedColors(source)
	return nil
}

type ResponseMediaItem struct {
	OccuredAt       string           `json:"occurred_at"`
	Title           string           `json:"title"`
	Subtitle        string           `json:"subtitle"`
	Category        string           `json:"category"`
	Source          string           `json:"source"`
	Image           string           `json:"image"`
	Duration        int              `json:"duration_ms"`
	DominantColours SerializedColors `json:"dominant_colours"`
}

type MediaItem struct {
	CreatedAt       int64            `json:"-"`
	Title           string           `json:"title"`
	Subtitle        string           `json:"subtitle"`
	Category        string           `json:"category"`
	IsActive        bool             `json:"is_active"`
	Elapsed         int              `json:"elapsed_ms"`
	Duration        int              `json:"duration_ms"`
	Source          string           `json:"source"`
	Image           string           `json:"image"`
	DominantColours SerializedColors `json:"dominant_colours"`
	Backfilled      bool             `json:"-"`
}

type SlackProfile struct {
	StatusText       string  `json:"status_text"`
	StatusEmoji      string  `json:"status_emoji"`
	StatusExpiration float64 `json:"status_expiration"`
}
