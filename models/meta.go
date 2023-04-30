package models

import (
	"database/sql/driver"
	"errors"
	"strings"
)

// SerializableColours is a custom DB extension type that stores
// a string slice as a comma separate value in the database
// Example input: []string{"#020304", "#6581be"}
// Example DB value: #020304,#6581be
type SerializableColours []string

func (s SerializableColours) Value() (driver.Value, error) {
	return strings.Join(s, ","), nil
}

func (s *SerializableColours) Scan(src interface{}) error {
	var source []string
	switch src.(type) {
	case string:
		source = strings.Split(src.(string), ",")
	default:
		return errors.New("incompatible type for SerializedColors")
	}
	*s = SerializableColours(source)
	return nil
}

type ResponseMediaItem struct {
	OccuredAt       string              `json:"occurred_at"`
	Title           string              `json:"title"`
	Subtitle        string              `json:"subtitle"`
	Category        string              `json:"category"`
	Source          string              `json:"source"`
	Image           string              `json:"image"`
	Duration        int                 `json:"duration_ms"`
	DominantColours SerializableColours `json:"dominant_colours"`
}

// Used in V4 but not renamed until V3 is deprecated
type ComboDBMediaItem struct {
	ID              uint                `json:"-" db:"id"`
	OccuredAt       int64               `json:"occurred_at" db:"created_at"`
	Title           string              `json:"title" db:"title"`
	Subtitle        string              `json:"subtitle" db:"subtitle"`
	Category        string              `json:"category" db:"category"`
	IsActive        bool                `json:"is_active" db:"is_active"`
	Elapsed         int                 `json:"elapsed_ms" db:"-"`
	Duration        int                 `json:"duration_ms" db:"duration_ms"`
	Source          string              `json:"source" db:"source"`
	Image           string              `json:"image" db:"image"`
	DominantColours SerializableColours `json:"dominant_colours" db:"dominant_colours"`
	Backfilled      bool                `json:"-" db:"-"`
}

type MediaItem struct {
	CreatedAt       int64               `json:"-"`
	Title           string              `json:"title"`
	Subtitle        string              `json:"subtitle"`
	Category        string              `json:"category"`
	IsActive        bool                `json:"is_active"`
	Elapsed         int                 `json:"elapsed_ms"`
	Duration        int                 `json:"duration_ms"`
	Source          string              `json:"source"`
	Image           string              `json:"image"`
	DominantColours SerializableColours `json:"dominant_colours"`
	Backfilled      bool                `json:"-"`
}
