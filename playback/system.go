package playback

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
)

type PlaybackSystem struct {
	state []FullPlaybackEntry
	db    *sqlx.DB
	m     sync.RWMutex
}

func NewPlaybackSystem(db *sqlx.DB) *PlaybackSystem {
	return &PlaybackSystem{
		state: []FullPlaybackEntry{},
		db:    db,
	}
}

func (ps *PlaybackSystem) GetState() []FullPlaybackEntry {
	ps.m.RLock()
	defer ps.m.RUnlock()
	state := make([]FullPlaybackEntry, len(ps.state))
	copy(state, ps.state)
	return state
}

func (ps *PlaybackSystem) UpdatePlaybackState(update Update) error {
	// Ensure we have an ID. It's deterministic so doesn't matter
	// if we run it a bunch of times
	update.MediaItem.ID = GenerateMediaID(&update)

	tx, err := ps.db.Beginx()
	if err != nil {
		return err
	}

	var committed bool
	var broadcast bool
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
		if broadcast {
			ps.broadcastEvent()
		}
	}()

	elapsed := int(update.Elapsed.Milliseconds())

	var existingEntry PlaybackEntry
	err = tx.Get(&existingEntry, `
	  SELECT id, media_id, elapsed, status, is_active, state_changed_at
	  FROM playback_entries
	  WHERE category = ? AND source = ?
	  ORDER BY updated_at DESC LIMIT 1`,
		update.MediaItem.Category, update.MediaItem.Source)

	if err == nil {
		slog.Debug("Found existing entry to update",
			slog.String("media_id", update.MediaItem.ID),
			slog.String("old_status", string(existingEntry.Status)),
			slog.String("new_status", string(update.Status)))
		// We have an active entry to compare our new state
		if existingEntry.MediaID != update.MediaItem.ID {
			// We now have a newly active entry so let's ensure the current one
			// is deactivated if it isn't already
			if existingEntry.IsActive {
				now := time.Now()
				_, err := tx.Exec(`
				  UPDATE playback_entries
				  SET is_active = FALSE, status = ?, updated_at = ?, state_changed_at = ?
				  WHERE id = ?`,
					StatusStopped, now, now, existingEntry.ID)
				if err != nil {
					return fmt.Errorf("failed to deactivate old entry: %+v", err)
				}
			}
		} else {
			if existingEntry.Status != update.Status || existingEntry.Elapsed != elapsed {
				now := time.Now()
				stateChangedAt := existingEntry.StateChangedAt
				if existingEntry.Status != update.Status {
					stateChangedAt = now
				}
				_, err := tx.Exec(`
				UPDATE playback_entries
				SET elapsed = ?, status = ?, is_active = ?, updated_at = ?, state_changed_at = ?
				WHERE id = ?`,
					elapsed, update.Status, update.Status == StatusPlaying, now, stateChangedAt, existingEntry.ID)
				if err != nil {
					return err
				}
				broadcast = true
			}

			slog.Debug("Updated existing entry", slog.String("media_id", update.MediaItem.ID))

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
		return fmt.Errorf("failed to insert new item: %+v", err)
	}

	// Now we can insert our playback entry and wrap up the update process
	now := time.Now()
	_, err = tx.Exec(`
	  INSERT INTO playback_entries
	  (media_id, category, created_at, elapsed, status, is_active, updated_at, state_changed_at, source)
	  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		update.MediaItem.ID, update.MediaItem.Category, now, elapsed, update.Status, update.Status == StatusPlaying, now, now, update.MediaItem.Source)
	if err != nil {
		return fmt.Errorf("failed to insert new playback entry: %+v", err)
	}

	slog.Debug("Inserted new playback entry", slog.String("media_id", update.MediaItem.ID))

	if err = tx.Commit(); err != nil {
		return err
	}
	broadcast = true
	committed = true
	return nil
}

type Event struct {
}

// CurrentPlayback is what a client should show right now: the active items, or when nothing is
// active, the most recent item so a pause or stop still has something to describe. The playing
// endpoint and the event stream both use it, so a client sees the same shape from either.
func (ps *PlaybackSystem) CurrentPlayback() ([]FullPlaybackEntry, error) {
	state := ps.GetState()
	if len(state) == 0 {
		var err error
		state, err = ps.GetHistory(1)
		if err != nil {
			return nil, err
		}
	}
	for i, entry := range state {
		state[i].Image = "/static/" + strings.ReplaceAll(entry.ID, ":", ".") + ".jpeg"
	}
	return state, nil
}

func (ps *PlaybackSystem) broadcastEvent() {
	current, err := ps.CurrentPlayback()
	if err != nil {
		slog.Error("Failed to load current playback for broadcast", slog.String("error", err.Error()))
		return
	}
	jsonState, _ := json.Marshal(current)
	events.Server.Publish(jsonState)
}

func (ps *PlaybackSystem) RefreshCurrentPlayback() error {
	entries, err := ps.GetActivePlayback()
	if err != nil {
		return err
	}

	ps.m.Lock()
	defer ps.m.Unlock()

	ps.state = entries

	return nil
}

// ps.state is expected to always be up to date
func (ps *PlaybackSystem) GetActivePlayback() ([]FullPlaybackEntry, error) {
	var results []FullPlaybackEntry

	err := ps.db.Select(&results, `
	  SELECT
	    m.id, m.title, m.subtitle, m.category, m.duration, m.source, m.image, m.dominant_colours,
		p.id as playback_id, p.created_at, p.elapsed, p.status, p.is_active, p.updated_at, p.state_changed_at
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
		p.id as playback_id, p.created_at, p.elapsed, p.status, p.is_active, p.updated_at, p.state_changed_at
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

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE playback_entries
		SET is_active = FALSE, status = ?, updated_at = ?, state_changed_at = ?
		WHERE is_active = TRUE AND media_id IN (
			SELECT id FROM media_items WHERE source = ?
		)
	`, StatusStopped, now, now, source)

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
		p.id as playback_id, p.created_at, p.elapsed, p.status, p.is_active, p.updated_at, p.state_changed_at
	  FROM media_items m
	  JOIN playback_entries p ON m.id = p.media_id
	  WHERE p.is_active = FALSE
	  ORDER BY p.state_changed_at DESC
	  LIMIT ?
	`, limit)

	return results, err
}

func (ps *PlaybackSystem) GetMediaItemByID(id string) (MediaItem, error) {
	var item MediaItem
	err := ps.db.Get(&item, `
	  SELECT id, title, subtitle, category, duration, source, image, dominant_colours
	  FROM media_items WHERE id = ?`, id)
	return item, err
}

func (ps *PlaybackSystem) ResolveCover(cfg config.Config, hash, imageURL string) (string, models.SerializableColours, error) {
	if existing, err := ps.GetMediaItemByID(hash); err == nil {
		return existing.Image, existing.DominantColours, nil
	}
	image, extension, domColours, err := utils.ExtractImageContent(imageURL)
	if err != nil {
		return "", nil, err
	}
	coverUrl, err := utils.SaveCover(cfg, hash, image, extension)
	if err != nil {
		return "", nil, err
	}
	return coverUrl, domColours, nil
}

func (ps *PlaybackSystem) HasPlaybackEntry(mediaID string) (bool, error) {
	var count int
	err := ps.db.Get(&count, `SELECT COUNT(*) FROM playback_entries WHERE media_id = ?`, mediaID)
	return count > 0, err
}

func (ps *PlaybackSystem) DeleteItem(playback_id int) error {
	if _, err := ps.db.Exec(`DELETE FROM playback_entries WHERE id = ?`, playback_id); err != nil {
		return err
	}
	return nil
}
