package main

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/models"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err)

	// embed is defined in main.go
	goose.SetBaseFS(embedMigrations)

	err = goose.SetDialect("sqlite3")
	require.NoError(t, err)

	err = goose.Up(db.DB, "migrations")
	require.NoError(t, err)

	return db
}

func TestPlaybackSystem_UpdatePlaybackState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	// 1. Persisting a new media item + playback entry
	category := string(Track)
	source := string(Plex)
	initialItemId := "plex:track:abc123"
	secondItemId := "spotify:track:abc123"

	update := PlaybackUpdate{
		MediaItem: MediaItem{
			ID:              initialItemId,
			Title:           "a good song",
			Subtitle:        "some artist",
			Category:        category,
			Duration:        180000,
			Source:          source,
			Image:           "https://example.com/blah.jpg",
			DominantColours: models.SerializableColours{"#abc123", "bcd234"},
		},
		Elapsed: 30 * time.Second,
		Status:  StatusPlaying,
	}

	err := ps.UpdatePlaybackState(update)
	assert.NoError(t, err)

	// 1a. Confirm that our media item is correct
	var mediaItem MediaItem
	err = db.Get(&mediaItem, "SELECT * FROM media_items WHERE id = ?", initialItemId)
	assert.NoError(t, err)
	assert.Equal(t, "a good song", mediaItem.Title)
	assert.Equal(t, "some artist", mediaItem.Subtitle)
	assert.Equal(t, category, mediaItem.Category)
	assert.Equal(t, 180000, mediaItem.Duration)
	assert.Equal(t, source, mediaItem.Source)
	assert.Equal(t, "https://example.com/blah.jpg", mediaItem.Image)
	assert.Equal(t, models.SerializableColours{"#abc123", "bcd234"}, mediaItem.DominantColours)

	// 1b. Confirm that our playback entry was inserted
	var playbackEntry PlaybackEntry
	err = db.Get(&playbackEntry, "SELECT * FROM playback_entries WHERE media_id = ?", initialItemId)
	assert.NoError(t, err)
	assert.Equal(t, initialItemId, playbackEntry.MediaID)
	assert.Equal(t, category, playbackEntry.Category)
	assert.Equal(t, 30000, playbackEntry.Elapsed)
	assert.Equal(t, StatusPlaying, playbackEntry.Status)
	assert.Equal(t, true, playbackEntry.IsActive)

	// 2. Update existing playback entry
	update.Elapsed = 60 * time.Second
	err = ps.UpdatePlaybackState(update)
	assert.NoError(t, err)

	err = db.Get(&playbackEntry, "SELECT * FROM playback_entries WHERE media_id = ?", initialItemId)
	assert.NoError(t, err)
	assert.Equal(t, 60000, playbackEntry.Elapsed)

	// 3. New item in same category should deactivate existing entry
	update2 := PlaybackUpdate{
		MediaItem: MediaItem{
			ID:              secondItemId,
			Title:           "a better song",
			Subtitle:        "another artist",
			Category:        category,
			Duration:        150000,
			Source:          "spotify", // not a real source currently but it doesn't matter
			Image:           "https://blah.net/c.png",
			DominantColours: models.SerializableColours{"#def345", "efg456"},
		},
		Elapsed: 18 * time.Second,
		Status:  StatusPlaying,
	}

	err = ps.UpdatePlaybackState(update2)
	assert.NoError(t, err)

	// 3a. Check that our initial entry is inactive
	err = db.Get(&playbackEntry, "SELECT * FROM playback_entries WHERE media_id = ?", initialItemId)
	assert.NoError(t, err)
	assert.Equal(t, false, playbackEntry.IsActive)

	// 3b. Check that our new entry is active
	err = db.Get(&playbackEntry, "SELECT * FROM playback_entries WHERE media_id = ?", secondItemId)
	assert.NoError(t, err)
	assert.Equal(t, true, playbackEntry.IsActive)
}

func TestPlaybackSystem_GetActivePlayback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	// Initial update should return one entry
	update := PlaybackUpdate{
		MediaItem: MediaItem{
			ID:              "plex:song:blah",
			Title:           "a song",
			Subtitle:        "artist",
			Category:        "track",
			Duration:        120000,
			Source:          "blah",
			Image:           "https://bleg.net",
			DominantColours: models.SerializableColours{"#abc123"},
		},
		Elapsed: 20 * time.Second,
		Status:  StatusPlaying,
	}
	err := ps.UpdatePlaybackState(update)
	require.NoError(t, err)

	activePlayback, err := ps.GetActivePlayback()
	assert.NoError(t, err)
	assert.Len(t, activePlayback, 1)
	assert.Equal(t, "plex:song:blah", activePlayback[0].ID)
	assert.Equal(t, "a song", activePlayback[0].Title)
	assert.Equal(t, "artist", activePlayback[0].Subtitle)
	assert.Equal(t, 20000, activePlayback[0].Elapsed)
	assert.Equal(t, 120000, activePlayback[0].Duration)
	assert.Equal(t, "blah", activePlayback[0].Source)
	assert.Equal(t, "https://bleg.net", activePlayback[0].Image)
	assert.Equal(t, models.SerializableColours{"#abc123"}, activePlayback[0].DominantColours)
	assert.Equal(t, true, activePlayback[0].IsActive)
}
