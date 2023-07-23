package jobs

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/h2non/gock"
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/stretchr/testify/assert"
)

func TestGetRecentlyReadManga_BadStatusCode(t *testing.T) {
	expected := models.MediaItem{}

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(anilistGraphqlEndpoint).
		Post("/").
		Reply(500)

	client := http.Client{}

	GetRecentlyReadManga(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetRecentlyReadManga_BadBody(t *testing.T) {
	expected := models.MediaItem{}

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(anilistGraphqlEndpoint).
		Post("/").
		Reply(200)

	client := http.Client{}

	GetRecentlyReadManga(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}

func TestGetRecentlyReadManga_EmptyBody(t *testing.T) {
	expected := models.MediaItem{}

	defer gock.Off()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	gock.New(anilistGraphqlEndpoint).
		Post("/").
		Reply(200).
		JSON(map[string]string{})

	client := http.Client{}

	GetRecentlyReadManga(sqlxDB, client)

	assert.Equal(t, expected, CurrentPlaybackItem)
}
