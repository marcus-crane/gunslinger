package main

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type MediaItem struct {
	ID       string `db:"id"`
	Title    string `db:"title"`
	Subtitle string `db:"subtitle"`
	Category string `db:"category"`
	Duration int    `db:"duration"` // milliseconds
	Source   string `db:"source"`
	Image    string `db:"image"`
}

type PlaybackStatus string

const (
	StatusPlaying PlaybackStatus = "playing"
	StatusPaused  PlaybackStatus = "paused"
	StatusStopped PlaybackStatus = "stopped"
)

type Category string

// TODO: Normalise gaming into videogame and podcast_episode into podcast(?)
const (
	Episode Category = "episode"
	Gaming  Category = "gaming"
	Manga   Category = "manga"
	Movie   Category = "movie"
	Podcast Category = "podcast_episode"
	Track   Category = "track"
)

type Source string

const (
	Anilist    Source = "anilist"
	Plex       Source = "plex"
	Steam      Source = "steam"
	Trakt      Source = "trakt"
	TraktCasts Source = "traktcasts"
)

type PlaybackEntry struct {
	ID        int            `db:"id"`
	MediaID   string         `db:"media_id"`
	Category  string         `db:"category"`
	StartedAt time.Time      `db:"started_at"`
	Elapsed   int            `db:"elapsed"` // milliseconds
	Status    PlaybackStatus `db:"status"`
	IsActive  bool           `db:"is_active"`
	UpdatedAt time.Time      `db:"updated_at"`
}

type FullPlaybackEntry struct {
	// MediaItem fields
	ID       string `db:"id" json:"id"`
	Title    string `db:"title" json:"title"`
	Subtitle string `db:"subtitle" json:"subtitle"`
	Category string `db:"category" json:"category"`
	Duration int    `db:"duration" json:"duration"`
	Source   string `db:"source" json:"source"`
	Image    string `db:"image" json:"image"`

	// PlaybackEntry fields
	PlaybackID int            `db:"playback_id" json:"-"`
	StartedAt  time.Time      `db:"started_at" json:"started_at"`
	Elapsed    int            `db:"elapsed" json:"elapsed"`
	Status     PlaybackStatus `db:"status" json:"status"`
	IsActive   bool           `db:"is_active" json:"is_active"`
	UpdatedAt  time.Time      `db:"updated_at" json:"updated_at"`
}

type PlaybackUpdate struct {
	MediaItem MediaItem
	Elapsed   time.Duration
	Status    PlaybackStatus
}

type PlaybackSystem struct {
	db *sqlx.DB
}

func (ps *PlaybackSystem) UpdatePlaybackState(update PlaybackUpdate) error {
	tx, err := ps.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	elapsedMs := int(update.Elapsed.Milliseconds())
	// TODO: Do we need to skip non-active stuff from being saved? Probably fine

	var existingEntry PlaybackEntry
	err = tx.Get(&existingEntry, `
	  SELECT id, media_id, elapsed, status
	  FROM playback_entries
	  WHERE category = ? AND is_active = TRUE`,
		update.MediaItem.Category)

	if err == nil {
		// We have an active entry to compare our new state to
		if existingEntry.MediaID != update.MediaItem.ID {
			// We now have a newly active entry so let's deactivate the current one
			_, err := tx.Exec(`
			  UPDATE playback_entries
			  SET is_active = FALSE, status = ?, updated_at = ?
			  WHERE id = ?`,
				StatusStopped, now, existingEntry.ID)
			if err != nil {
				return err
			}
		} else {
			// The same media item has been seen again so we'll save the updated state
			_, err := tx.Exec(`
			  UPDATE playback_entries
			  SET elapsed = ?, status = ?, updated_at = ?
			  WHERE id = ?`,
				elapsedMs, update.Status, now, existingEntry.ID)
			if err != nil {
				return err
			}
			// We already know this item is saved since it's in progress so we can
			// bail out of our transaction early
			return tx.Commit()
		}
		// TODO: sqlx variant of NoRows?
	} else if err != sql.ErrNoRows {
		return err
	}

	// At this point, we're dealing with a new playback entry after deactivating the previous
	// one so we need to save metadata about this item to our database. If the item already
	// exists (ie; we've played it before) then we don't care, a no-op is perfectly fine.
	_, err = tx.NamedExec(`
	  INSERT INTO media_items
	  (id, title, subtitle, category, duration, source, image)
	  VALUES (:id, :title, :subtitle, :category, :duration, :source, :image)
	  ON CONFLICT (id) DO NOTHING`,
		update.MediaItem)
	if err != nil {
		return err
	}

	// Now we can insert our playback entry and wrap up the update process
	_, err = tx.Exec(`
	  INSERT INTO playback_entries
	  (media_id, category, started_at, elapsed, status, is_active, updated_at)
	  VALUES (?, ?, ?, ?, ?, ?, ?)`,
		update.MediaItem.ID, update.MediaItem.Category, now, elapsedMs, update.Status, true, now)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (ps *PlaybackSystem) GetActivePlayback() ([]FullPlaybackEntry, error) {
	var results []FullPlaybackEntry

	err := ps.db.Select(&results, `
	  SELECT
	    m.id, m.title, m.subtitle, m.category, m.duration, m.source, m.image,
		p.id as playback_id, p.started_at, p.elapsed, p.status, p.is_active, p.updated_at
	  FROM media_items m
	  JOIN playback_entries p ON m.id = p.media_id
	  WHERE p.is_active = TRUE
	  ORDER BY p.updated_at DESC
	`)

	return results, err
}
