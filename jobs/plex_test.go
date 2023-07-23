package jobs

import (
	"bytes"
	"net/http"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/h2non/gock"
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/stretchr/testify/assert"
)

func TestGetCurrentlyPlayingPlex_BadStatusCode(t *testing.T) {
	expected := models.MediaItem{}

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(500)

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingPlex_BadBody(t *testing.T) {
	expected := models.MediaItem{}

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(200)

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingPlex_EmptyBody(t *testing.T) {
	expected := models.MediaItem{}

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(200).
		JSON(map[string]string{})

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingPlex_NoPlayingItem(t *testing.T) {
	expected := models.MediaItem{
		IsActive: false,
		Source:   "plex",
	}

	events.Init()

	CurrentPlaybackItem.IsActive = true
	CurrentPlaybackItem.Source = "plex"

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.PlexResponse{
		MediaContainer: models.MediaContainer{
			Size: 0,
		},
	}

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(200).
		JSON(resp)

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingPlex_PlayingItemClip(t *testing.T) {
	expected := models.MediaItem{
		IsActive: false,
		Source:   "plex",
	}

	events.Init()

	CurrentPlaybackItem.IsActive = false
	CurrentPlaybackItem.Source = "plex"

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.PlexResponse{
		MediaContainer: models.MediaContainer{
			Size: 1,
			Metadata: []models.Metadata{
				{
					Type: "clip",
					Player: models.Player{
						State: "playing",
					},
				},
			},
		},
	}

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(200).
		JSON(resp)

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingPlex_PlayingItemEpisode(t *testing.T) {
	expected := models.MediaItem{
		Title:    "02x08 The Mistake",
		Subtitle: "House",
		Category: "episode",
		Elapsed:  1000,
		Duration: 2640888,
		IsActive: true,
		Source:   "plex",
		Image:    "/static/cover.e06f49aa-27d0-060d-6202-5652179aff16.jpeg",
		DominantColours: models.SerializableColours{
			"#514e29",
			"#d9d2c4",
			"#a59863",
			"#93653f",
			"#6d8b43",
			"#6dabdf",
		},
	}

	events.Init()

	CurrentPlaybackItem.IsActive = false
	CurrentPlaybackItem.Source = "plex"

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.PlexResponse{
		MediaContainer: models.MediaContainer{
			Size: 1,
			Metadata: []models.Metadata{
				{
					ParentIndex:      2,
					Index:            8,
					Title:            "The Mistake",
					GrandparentTitle: "House",
					Type:             "episode",
					Duration:         2640888,
					ViewOffset:       1000,
					Thumb:            "/images/thumb.jpg",
					Player: models.Player{
						State: "playing",
					},
				},
			},
		},
	}

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(200).
		JSON(resp)

	image, err := os.ReadFile("../fixtures/dog.jpg")
	if err != nil {
		panic("Failed to load in fixture")
	}

	gock.New(buildPlexURL("/images/thumb.jpg")).
		Get("/").
		Reply(200).
		Body(bytes.NewReader(image))

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingPlex_PlayingItemMovie(t *testing.T) {
	expected := models.MediaItem{
		Title:    "Heat",
		Subtitle: "Michael Mann",
		Category: "movie",
		Elapsed:  1000,
		Duration: 2640888,
		IsActive: true,
		Source:   "plex",
		Image:    "/static/cover.e06f49aa-27d0-060d-6202-5652179aff16.jpeg",
		DominantColours: models.SerializableColours{
			"#514e29",
			"#d9d2c4",
			"#a59863",
			"#93653f",
			"#6d8b43",
			"#6dabdf",
		},
	}

	events.Init()

	CurrentPlaybackItem.IsActive = false
	CurrentPlaybackItem.Source = "plex"

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.PlexResponse{
		MediaContainer: models.MediaContainer{
			Size: 1,
			Metadata: []models.Metadata{
				{
					Title:            "Heat",
					GrandparentTitle: "Michael Mann",
					Type:             "movie",
					Duration:         2640888,
					ViewOffset:       1000,
					Thumb:            "/images/thumb.jpg",
					Player: models.Player{
						State: "playing",
					},
					Director: []models.Director{
						{
							Name: "Michael Mann",
						},
					},
				},
			},
		},
	}

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(200).
		JSON(resp)

	image, err := os.ReadFile("../fixtures/dog.jpg")
	if err != nil {
		panic("Failed to load in fixture")
	}

	gock.New(buildPlexURL("/images/thumb.jpg")).
		Get("/").
		Reply(200).
		Body(bytes.NewReader(image))

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingPlex_PlayingItemTrack(t *testing.T) {
	expected := models.MediaItem{
		Title:    "DANGANRONPA SUPER MIX",
		Subtitle: "高田雅史",
		Category: "track",
		Elapsed:  1000,
		Duration: 2640888,
		IsActive: true,
		Source:   "plex",
		Image:    "/static/cover.e06f49aa-27d0-060d-6202-5652179aff16.jpeg",
		DominantColours: models.SerializableColours{
			"#514e29",
			"#d9d2c4",
			"#a59863",
			"#93653f",
			"#6d8b43",
			"#6dabdf",
		},
	}

	events.Init()

	CurrentPlaybackItem.IsActive = false
	CurrentPlaybackItem.Source = "plex"

	t.Setenv("PLEX_URL", "http://example.com")
	t.Setenv("PLEX_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.PlexResponse{
		MediaContainer: models.MediaContainer{
			Size: 1,
			Metadata: []models.Metadata{
				{
					Title:            "DANGANRONPA SUPER MIX",
					GrandparentTitle: "高田雅史",
					Type:             "track",
					Duration:         2640888,
					ViewOffset:       1000,
					ParentThumb:      "/images/thumb.jpg",
					Player: models.Player{
						State: "playing",
					},
				},
			},
		},
	}

	gock.New(buildPlexURL(plexSessionEndpoint)).
		Get("/").
		Reply(200).
		JSON(resp)

	image, err := os.ReadFile("../fixtures/dog.jpg")
	if err != nil {
		panic("Failed to load in fixture")
	}

	gock.New(buildPlexURL("/images/thumb.jpg")).
		Get("/").
		Reply(200).
		Body(bytes.NewReader(image))

	client := http.Client{}

	GetCurrentlyPlayingPlex(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}
