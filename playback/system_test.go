package playback

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/migrations"
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
	goose.SetBaseFS(migrations.GetMigrations())

	err = goose.SetDialect("sqlite3")
	require.NoError(t, err)

	err = goose.Up(db.DB, ".")
	require.NoError(t, err)

	// Gross, PlaybackSystem should handle this
	events.Init()

	return db
}

func TestPlaybackSystem_UpdatePlaybackState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	assert.Len(t, ps.State, 0)

	// 1. Persisting a new media item + playback entry
	category := string(Track)
	source := string(Plex)

	update := Update{
		MediaItem: MediaItem{
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
	initialItemId := GenerateMediaID(&update)

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

	// 1c. Confirm that the PlaybackSystem has fresh state
	assert.Len(t, ps.State, 1)
	assert.Equal(t, ps.State[0].Title, mediaItem.Title)
	assert.Equal(t, ps.State[0].Elapsed, 30000)

	// 2. Update existing playback entry
	update.Elapsed = 60 * time.Second
	update.Status = StatusPaused
	err = ps.UpdatePlaybackState(update)
	assert.NoError(t, err)

	err = db.Get(&playbackEntry, "SELECT * FROM playback_entries WHERE media_id = ?", initialItemId)
	assert.NoError(t, err)

	// 2a. Confirm that PlaybackSystem state is updated
	assert.Len(t, ps.State, 0)
	assert.Equal(t, 60000, playbackEntry.Elapsed)
	assert.Equal(t, false, playbackEntry.IsActive)

	// 3. New item in same category should deactivate existing entry
	update2 := Update{
		MediaItem: MediaItem{
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
	secondItemId := GenerateMediaID(&update2)

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

	// 3c. Check that PlaybackSystem has updated
	assert.Len(t, ps.State, 1)
	assert.Equal(t, ps.State[0].Title, "a better song")
}

func TestPlaybackUpdate_GenerateMediaID(t *testing.T) {
	// Initial update should return one entry
	update := Update{
		MediaItem: MediaItem{
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
	update.MediaItem.ID = GenerateMediaID(&update)
	assert.Equal(t, "blah:track:10755785467225436000", update.MediaItem.ID)
}

func TestPlaybackSystem_GetActivePlayback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	// Initial update should return one entry
	update := Update{
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
	assert.Equal(t, GenerateMediaID(&update), activePlayback[0].ID)
	assert.Equal(t, "a song", activePlayback[0].Title)
	assert.Equal(t, "artist", activePlayback[0].Subtitle)
	assert.Equal(t, 20000, activePlayback[0].Elapsed)
	assert.Equal(t, 120000, activePlayback[0].Duration)
	assert.Equal(t, "blah", activePlayback[0].Source)
	assert.Equal(t, "https://bleg.net", activePlayback[0].Image)
	assert.Equal(t, models.SerializableColours{"#abc123"}, activePlayback[0].DominantColours)
	assert.Equal(t, true, activePlayback[0].IsActive)
}

func TestPlaybackSystem_GetActivePlaybackBySource(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	update := Update{
		MediaItem: MediaItem{
			Title:           "a song",
			Subtitle:        "artist",
			Category:        "track",
			Duration:        120000,
			Source:          "plex",
			Image:           "https://bleg.net",
			DominantColours: models.SerializableColours{"#abc123"},
		},
		Elapsed: 20 * time.Second,
		Status:  StatusPlaying,
	}
	err := ps.UpdatePlaybackState(update)
	require.NoError(t, err)

	sourcePlayback, err := ps.GetActivePlaybackBySource(string(Plex))
	assert.NoError(t, err)
	assert.Len(t, sourcePlayback, 1)
	assert.Equal(t, GenerateMediaID(&update), sourcePlayback[0].ID)
	assert.Equal(t, "plex", sourcePlayback[0].Source)

	update2 := Update{
		MediaItem: MediaItem{
			Title:           "a better song",
			Subtitle:        "another artist",
			Category:        "track",
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

	// We expect that our Plex song is still active as we query playback by category and source
	sourcePlayback, err = ps.GetActivePlaybackBySource(string(Plex))
	assert.NoError(t, err)
	assert.Len(t, sourcePlayback, 1)

	// Mark our Plex song as active once again
	err = ps.UpdatePlaybackState(update)
	assert.NoError(t, err)

	update3 := Update{
		MediaItem: MediaItem{
			Title:           "wobbledogs",
			Subtitle:        "game maker",
			Category:        "game",
			Duration:        0, // Games don't have a duration
			Source:          "steam",
			Image:           "https://blah.net/c.png",
			DominantColours: models.SerializableColours{"#def345", "efg456"},
		},
		Elapsed: 0,
		Status:  StatusPlaying,
	}

	// Now start up a Steam game
	err = ps.UpdatePlaybackState(update3)
	assert.NoError(t, err)

	// We should expect that our Plex song is still active
	sourcePlayback, err = ps.GetActivePlaybackBySource(string(Plex))
	assert.NoError(t, err)
	assert.Len(t, sourcePlayback, 1)
	assert.Equal(t, GenerateMediaID(&update), sourcePlayback[0].ID)
	assert.Equal(t, "plex", sourcePlayback[0].Source)
}

func TestPlaybackSystem_DeactivateBySource(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	update := Update{
		MediaItem: MediaItem{
			Title:           "a song",
			Subtitle:        "artist",
			Category:        "track",
			Duration:        120000,
			Source:          "plex",
			Image:           "https://bleg.net",
			DominantColours: models.SerializableColours{"#abc123"},
		},
		Elapsed: 20 * time.Second,
		Status:  StatusPlaying,
	}
	err := ps.UpdatePlaybackState(update)
	require.NoError(t, err)

	update2 := Update{
		MediaItem: MediaItem{
			Title:           "action movie",
			Subtitle:        "directed by person",
			Category:        "movie",
			Duration:        999999,
			Source:          "plex",
			Image:           "https://example.com/movie.jpg",
			DominantColours: models.SerializableColours{"#abc123"},
		},
		Elapsed: 60 * time.Minute,
		Status:  StatusPlaying,
	}
	err = ps.UpdatePlaybackState(update2)
	require.NoError(t, err)

	update3 := Update{
		MediaItem: MediaItem{
			Title:           "wobbledogs",
			Subtitle:        "game maker",
			Category:        "game",
			Duration:        0, // Games don't have a duration
			Source:          "steam",
			Image:           "https://blah.net/c.png",
			DominantColours: models.SerializableColours{"#def345", "efg456"},
		},
		Elapsed: 0,
		Status:  StatusPlaying,
	}

	err = ps.UpdatePlaybackState(update3)
	assert.NoError(t, err)

	sourcePlayback, err := ps.GetActivePlaybackBySource(string(Plex))
	assert.NoError(t, err)
	assert.Len(t, sourcePlayback, 2)
	assert.Equal(t, GenerateMediaID(&update2), sourcePlayback[0].ID)
	assert.Equal(t, "plex", sourcePlayback[0].Source)
	assert.Equal(t, GenerateMediaID(&update), sourcePlayback[1].ID)
	assert.Equal(t, "plex", sourcePlayback[1].Source)

	err = ps.DeactivateBySource(string(Plex))
	assert.NoError(t, err)

	sourcePlayback, err = ps.GetActivePlaybackBySource(string(Plex))
	assert.NoError(t, err)
	assert.Len(t, sourcePlayback, 0)

	activePlayback, err := ps.GetActivePlayback()
	assert.NoError(t, err)
	assert.Equal(t, GenerateMediaID(&update3), activePlayback[0].ID)
	assert.Equal(t, "steam", activePlayback[0].Source)
}

func TestPlaybackSystem_GetHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	update := Update{
		MediaItem: MediaItem{
			Title:           "a song",
			Subtitle:        "artist",
			Category:        "track",
			Duration:        120000,
			Source:          "blah",
			Image:           "https://bleg.net",
			DominantColours: models.SerializableColours{"#abc123"},
		},
		Elapsed: 20 * time.Second,
		Status:  StatusPaused,
	}
	err := ps.UpdatePlaybackState(update)
	require.NoError(t, err)

	history, err := ps.GetHistory(1)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, GenerateMediaID(&update), history[0].ID)
	assert.Equal(t, "a song", history[0].Title)
	assert.Equal(t, "artist", history[0].Subtitle)
	assert.Equal(t, 20000, history[0].Elapsed)
	assert.Equal(t, 120000, history[0].Duration)
	assert.Equal(t, "blah", history[0].Source)
	assert.Equal(t, "https://bleg.net", history[0].Image)
	assert.Equal(t, models.SerializableColours{"#abc123"}, history[0].DominantColours)
	assert.Equal(t, false, history[0].IsActive)

	update2 := Update{
		MediaItem: MediaItem{
			Title:           "a better song",
			Subtitle:        "another artist",
			Category:        "track",
			Duration:        150000,
			Source:          "spotify", // not a real source currently but it doesn't matter
			Image:           "https://blah.net/c.png",
			DominantColours: models.SerializableColours{"#def345", "efg456"},
		},
		Elapsed: 18 * time.Second,
		Status:  StatusStopped,
	}

	err = ps.UpdatePlaybackState(update2)
	assert.NoError(t, err)

	history, err = ps.GetHistory(10)
	assert.NoError(t, err)
	assert.Len(t, history, 2)

	// We expect newly updated songs to be returned first
	assert.Equal(t, GenerateMediaID(&update2), history[0].ID)
	assert.Equal(t, GenerateMediaID(&update), history[1].ID)

	_, err = ps.GetHistory(0)
	assert.Error(t, err)

	_, err = ps.GetHistory(-1)
	assert.Error(t, err)
}

func TestPlaybackSystem_DeleteItem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := &PlaybackSystem{db: db}

	update := Update{
		MediaItem: MediaItem{
			Title:           "a song",
			Subtitle:        "artist",
			Category:        "track",
			Duration:        120000,
			Source:          "blah",
			Image:           "https://bleg.net",
			DominantColours: models.SerializableColours{"#abc123"},
		},
		Elapsed: 20 * time.Second,
		Status:  StatusPaused,
	}
	err := ps.UpdatePlaybackState(update)
	require.NoError(t, err)

	history, err := ps.GetHistory(1)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, GenerateMediaID(&update), history[0].ID)
	assert.Equal(t, "a song", history[0].Title)
	assert.Equal(t, "artist", history[0].Subtitle)
	assert.Equal(t, 20000, history[0].Elapsed)
	assert.Equal(t, 120000, history[0].Duration)
	assert.Equal(t, "blah", history[0].Source)
	assert.Equal(t, "https://bleg.net", history[0].Image)
	assert.Equal(t, models.SerializableColours{"#abc123"}, history[0].DominantColours)
	assert.Equal(t, false, history[0].IsActive)

	ps.DeleteItem(history[0].PlaybackID)

	history, err = ps.GetHistory(1)
	assert.Len(t, history, 0)
}
