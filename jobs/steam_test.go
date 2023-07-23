package jobs

import (
	"bytes"
	"fmt"
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

func TestGetCurrentlyPlayingSteam_BadStatusCode(t *testing.T) {
	expected := models.MediaItem{}

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Post("/").
		Reply(500)

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingSteam_BadBody(t *testing.T) {
	expected := models.MediaItem{}

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Post("/").
		Reply(200)

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingSteam_EmptyBody(t *testing.T) {
	expected := models.MediaItem{}

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Post("/").
		Reply(200).
		JSON(map[string]string{})

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingSteam_NoActiveGames(t *testing.T) {
	expected := models.MediaItem{}

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.SteamResponse{
		Players: []models.SteamUser{},
	}

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Get("/").
		Reply(200).
		JSON(resp)

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingSteam_MarkInactive(t *testing.T) {
	expected := models.MediaItem{
		IsActive: false,
		Source:   "steam",
	}

	CurrentPlaybackItem.IsActive = true
	CurrentPlaybackItem.Source = "steam"

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.SteamResponse{}

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Get("/").
		Reply(200).
		JSON(resp)

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingSteam_EmptyGameID(t *testing.T) {
	expected := models.MediaItem{
		IsActive: false,
		Source:   "steam",
	}

	CurrentPlaybackItem.IsActive = true
	CurrentPlaybackItem.Source = "steam"

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.SteamResponse{
		Players: []models.SteamUser{
			{
				GameID:       "",
				PersonaState: 4,
			},
		},
	}

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Get("/").
		Reply(200).
		JSON(resp)

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingSteam_PlayingGame(t *testing.T) {
	expected := models.MediaItem{
		Title:    "Final Fantasy XIV",
		Subtitle: "Square Enix",
		Category: "gaming",
		IsActive: true,
		Source:   "steam",
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

	CurrentPlaybackItem.IsActive = true
	CurrentPlaybackItem.Source = "steam"

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.SteamPlayerSummary{
		Response: models.SteamResponse{
			Players: []models.SteamUser{
				{
					GameID:       "abc123",
					PersonaState: 4,
				},
			},
		},
	}

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Get("/").
		Reply(200).
		JSON(resp)

	image, err := os.ReadFile("../fixtures/dog.jpg")
	if err != nil {
		panic("Failed to load in fixture")
	}

	gock.New("http://example.com/cover.jpg").
		Get("/").
		Reply(200).
		Body(bytes.NewReader(image))

	gameResp := models.SteamAppResponse{
		Data: models.SteamAppDetail{
			Name:        "Final Fantasy XIV",
			Developers:  []string{"Square Enix"},
			HeaderImage: "http://example.com/cover.jpg",
		},
	}

	gock.New(fmt.Sprintf(gameDetailEndpoint, "abc123")).
		Get("/").
		Reply(200).
		JSON(map[string]models.SteamAppResponse{
			"abc123": gameResp,
		})

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetCurrentlyPlayingSteam_UnknownDeveloper(t *testing.T) {
	expected := models.MediaItem{
		Title:    "Custom Title",
		Subtitle: "Unknown Developer",
		Category: "gaming",
		IsActive: true,
		Source:   "steam",
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

	CurrentPlaybackItem.IsActive = true
	CurrentPlaybackItem.Source = "steam"

	t.Setenv("STEAM_TOKEN", "123")

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	resp := models.SteamPlayerSummary{
		Response: models.SteamResponse{
			Players: []models.SteamUser{
				{
					GameID:       "abc123",
					PersonaState: 4,
				},
			},
		},
	}

	gock.New(fmt.Sprintf(profileEndpoint, "123")).
		Get("/").
		Reply(200).
		JSON(resp)

	image, err := os.ReadFile("../fixtures/dog.jpg")
	if err != nil {
		panic("Failed to load in fixture")
	}

	gock.New("http://example.com/cover.jpg").
		Get("/").
		Reply(200).
		Body(bytes.NewReader(image))

	gameResp := models.SteamAppResponse{
		Data: models.SteamAppDetail{
			Name:        "Custom Title",
			Developers:  []string{},
			HeaderImage: "http://example.com/cover.jpg",
		},
	}

	gock.New(fmt.Sprintf(gameDetailEndpoint, "abc123")).
		Get("/").
		Reply(200).
		JSON(map[string]models.SteamAppResponse{
			"abc123": gameResp,
		})

	client := http.Client{}

	GetCurrentlyPlayingSteam(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}
