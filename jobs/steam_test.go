package jobs

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/h2non/gock"
	"github.com/jmoiron/sqlx"
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
