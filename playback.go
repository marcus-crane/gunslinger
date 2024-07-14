package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/models"
)

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

// PlaybackEntry is a unique instance of a piece of media being played. If a movie is watched 5 times,
// there will be one MediaItem entry with five unique PlaybackEntry instances. PlaybackEntry instances
// may be "revived" such as if a podcast is paused and then picked up again the next day. Once completed,
// a PlaybackEntry should not be reused though.
type PlaybackEntry struct {
	ID        int            `db:"id"`
	MediaID   string         `db:"media_id"`
	Category  string         `db:"category"`
	CreatedAt time.Time      `db:"created_at"`
	Elapsed   int            `db:"elapsed"` // milliseconds
	Status    PlaybackStatus `db:"status"`
	IsActive  bool           `db:"is_active"`
	UpdatedAt time.Time      `db:"updated_at"`
}

// MediaItem stores metadata about each piece of media that is played ie; movies, tv series, games
// It's generic enough to support differences in media types such as music tracks needing a title,
// album name and artist while a game may need a title, developer name and year. Currently, each
// media source scraper is responsible for constructing the appropriate titles such as joining
// a movie name and year into a title field. In future, an explicit author field may be added.
type MediaItem struct {
	ID              string                     `db:"id"`
	Title           string                     `db:"title"`
	Subtitle        string                     `db:"subtitle"`
	Category        string                     `db:"category"`
	Duration        int                        `db:"duration"`
	Source          string                     `db:"source"`
	Image           string                     `db:"image"`
	DominantColours models.SerializableColours `db:"dominant_colours"`
}

// FullPlaybackEntry reflects a single PlaybackEntry with MediaItem metadata attached
// in order to power any clients that want to render full playback info.
type FullPlaybackEntry struct {
	// MediaItem fields
	ID              string                     `db:"id" json:"id"`
	Title           string                     `db:"title" json:"title"`
	Subtitle        string                     `db:"subtitle" json:"subtitle"`
	Category        string                     `db:"category" json:"category"`
	Duration        int                        `db:"duration" json:"duration_ms"` // TODO: Drop _ms suffix
	Source          string                     `db:"source" json:"source"`
	Image           string                     `db:"image" json:"image"` // TODO: Construct image URL from media_id
	DominantColours models.SerializableColours `db:"dominant_colours" json:"dominant_colours"`

	// PlaybackEntry fields
	PlaybackID int            `db:"playback_id" json:"-"`
	CreatedAt  time.Time      `db:"created_at" json:"created_at"`
	Elapsed    int            `db:"elapsed" json:"elapsed_ms"` // TODO: Drop _ms suffix
	Status     PlaybackStatus `db:"status" json:"status"`
	IsActive   bool           `db:"is_active" json:"is_active"`
	UpdatedAt  time.Time      `db:"updated_at" json:"updated_at"`
}

type PlaybackUpdate struct {
	MediaItem MediaItem
	Elapsed   time.Duration
	Status    PlaybackStatus
}

func GenerateMediaID(p *PlaybackUpdate) string {
	hashString := fmt.Sprintf("%s-%s-%s-%d-%s-%s",
		p.MediaItem.Title,
		p.MediaItem.Subtitle,
		p.MediaItem.Category,
		p.MediaItem.Duration,
		p.MediaItem.Source,
		p.MediaItem.Image,
	)
	return fmt.Sprintf(
		"%s:%s:%d",
		p.MediaItem.Source,
		p.MediaItem.Category,
		xxhash.Sum64String(hashString),
	)
}

type PlaybackSystem struct {
	State []FullPlaybackEntry
	db    *sqlx.DB
	m     sync.RWMutex
}

func NewPlaybackSystem(db *sqlx.DB) *PlaybackSystem {
	return &PlaybackSystem{
		State: []FullPlaybackEntry{},
		db:    db,
	}
}

func (ps *PlaybackSystem) UpdatePlaybackState(update PlaybackUpdate) error {
	// Ensure we have an ID. It's deterministic so doesn't matter
	// if we run it a bunch of times
	update.MediaItem.ID = GenerateMediaID(&update)

	tx, err := ps.db.Beginx()
	if err != nil {
		return err
	}

	var committed bool
	defer func() {
		if !committed {
			tx.Rollback()
		} else {
			ps.RefreshCurrentPlayback()
			// TODO: Publish update to clients
			// byteStream := new(bytes.Buffer)
			// json.NewEncoder(byteStream).Encode(update)
			// events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		}
	}()

	elapsed := int(update.Elapsed.Milliseconds())
	// TODO: Do we need to skip non-active stuff from being saved? Probably fine

	var existingEntry PlaybackEntry
	err = tx.Get(&existingEntry, `
	  SELECT id, media_id, elapsed, status
	  FROM playback_entries
	  WHERE category = ? AND is_active = TRUE`,
		update.MediaItem.Category)

	if err == nil {
		// We have an active entry to compare our new state
		if existingEntry.MediaID != update.MediaItem.ID {
			// We now have a newly active entry so let's deactivate the current one
			_, err := tx.Exec(`
			  UPDATE playback_entries
			  SET is_active = FALSE, status = ?, updated_at = ?
			  WHERE id = ?`,
				StatusStopped, time.Now(), existingEntry.ID)
			if err != nil {
				return err
			}
		} else {
			// The same media item has been seen again so we'll save the updated state
			_, err := tx.Exec(`
			  UPDATE playback_entries
			  SET elapsed = ?, status = ?, updated_at = ?
			  WHERE id = ?`,
				elapsed, update.Status, time.Now(), existingEntry.ID)
			if err != nil {
				return err
			}
			// We already know this item is saved since it's in progress so we can
			// bail out of our transaction early
			if err = tx.Commit(); err != nil {
				return err
			}
			committed = true
			return nil
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
	  (id, title, subtitle, category, duration, source, image, dominant_colours)
	  VALUES (:id, :title, :subtitle, :category, :duration, :source, :image, :dominant_colours)
	  ON CONFLICT (id) DO NOTHING`,
		update.MediaItem)
	if err != nil {
		return err
	}

	// Now we can insert our playback entry and wrap up the update process
	_, err = tx.Exec(`
	  INSERT INTO playback_entries
	  (media_id, category, created_at, elapsed, status, is_active, updated_at)
	  VALUES (?, ?, ?, ?, ?, ?, ?)`,
		update.MediaItem.ID, update.MediaItem.Category, time.Now(), elapsed, update.Status, true, time.Now())
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (ps *PlaybackSystem) RefreshCurrentPlayback() error {
	entries, err := ps.GetActivePlayback()
	if err != nil {
		return err
	}

	ps.m.Lock()
	defer ps.m.Unlock()

	ps.State = entries

	return nil
}

// ps.State is expected to always be up to date
func (ps *PlaybackSystem) GetActivePlayback() ([]FullPlaybackEntry, error) {
	var results []FullPlaybackEntry

	err := ps.db.Select(&results, `
	  SELECT
	    m.id, m.title, m.subtitle, m.category, m.duration, m.source, m.image, m.dominant_colours,
		p.id as playback_id, p.created_at, p.elapsed, p.status, p.is_active, p.updated_at
	  FROM media_items m
	  JOIN playback_entries p ON m.id = p.media_id
	  WHERE p.is_active = TRUE
	  ORDER BY p.updated_at DESC
	`)

	return results, err
}

func (ps *PlaybackSystem) GetActivePlaybackBySource(source string) ([]FullPlaybackEntry, error) {
	var results []FullPlaybackEntry

	err := ps.db.Select(&results, `
	  SELECT
	    m.id, m.title, m.subtitle, m.category, m.duration, m.source, m.image, m.dominant_colours,
		p.id as playback_id, p.created_at, p.elapsed, p.status, p.is_active, p.updated_at
	  FROM media_items m
	  JOIN playback_entries p ON m.id = p.media_id
	  WHERE p.is_active = TRUE AND m.source = ?
	  ORDER BY p.updated_at DESC
	`, source)

	return results, err
}

func (ps *PlaybackSystem) DeactivateBySource(source string) error {
	tx, err := ps.db.Beginx()
	if err != nil {
		return err
	}

	var committed bool
	defer func() {
		if !committed {
			tx.Rollback()
		} else {
			ps.RefreshCurrentPlayback()
		}
	}()

	_, err = tx.Exec(`
		UPDATE playback_entries
		SET is_active = FALSE, status = ?, updated_at = ?
		WHERE is_active = TRUE AND media_id IN (
			SELECT id FROM media_items WHERE source = ?
		)
	`, StatusStopped, time.Now(), source)

	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (ps *PlaybackSystem) GetHistory(limit int) ([]FullPlaybackEntry, error) {
	var results []FullPlaybackEntry

	if limit <= 0 {
		return results, fmt.Errorf("must request at least one historical item")
	}

	err := ps.db.Select(&results, `
	  SELECT
	    m.id, m.title, m.subtitle, m.category, m.duration, m.source, m.image, m.dominant_colours,
		p.id as playback_id, p.created_at, p.elapsed, p.status, p.is_active, p.updated_at
	  FROM media_items m
	  JOIN playback_entries p ON m.id = p.media_id
	  ORDER BY p.updated_at DESC
	  LIMIT ?
	`, limit)

	return results, err
}
