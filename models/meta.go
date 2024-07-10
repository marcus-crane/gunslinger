package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/cespare/xxhash/v2"
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
		if src != "" {
			source = strings.Split(src.(string), ",")
		}
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
	Hash            uint64              `json:"hash"`
	DominantColours SerializableColours `json:"dominant_colours"`
}

// Used in V4 but not renamed until V3 is deprecated
type ComboDBMediaItem struct {
	ID              uint                `json:"id" db:"id"`
	Hash            uint64              `json:"hash" db:"-"`
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

func GenerateHash(c ComboDBMediaItem) uint64 {
	hash := fmt.Sprintf(
		"%s-%s-%s-%t-%d-%d-%s-%s",
		c.Title,
		c.Subtitle,
		c.Category,
		c.IsActive,
		c.Elapsed,
		c.Duration,
		c.Source,
		c.Image,
	)
	return xxhash.Sum64String(hash)
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
	Hash            uint64              `json:"hash"`
	Backfilled      bool                `json:"-"`
}

func (m MediaItem) GenerateHash() uint64 {
	hash := fmt.Sprintf(
		"%s-%s-%s-%t-%d-%d-%s-%s",
		m.Title,
		m.Subtitle,
		m.Category,
		m.IsActive,
		m.Elapsed,
		m.Duration,
		m.Source,
		m.Image,
	)
	return xxhash.Sum64String(hash)
}
