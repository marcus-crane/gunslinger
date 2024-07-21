package db

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/models"
)

func TestPostgresStore_GetRecent(t *testing.T) {
	t.Parallel()

	query := "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 7"
	rows := sqlmock.NewRows([]string{"id", "created_at", "title", "subtitle", "category", "is_active", "duration_ms", "source", "image", "dominant_colours"}).
		AddRow(1, 0, "blah", "", "", false, 0, "", "", models.SerializableColours{}).
		AddRow(2, 0, "bleh", "", "", false, 0, "", "", models.SerializableColours{})

	p := fakePostgresStore(t, query, rows)
	want := []models.ComboDBMediaItem{
		{
			ID:    "1",
			Title: "blah",
		},
		{
			ID:    "2",
			Title: "bleh",
		},
	}
	got, err := p.GetRecent()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestPostgresStore_GetNewest(t *testing.T) {
	t.Parallel()

	query := "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 1"
	rows := sqlmock.NewRows([]string{"id", "created_at", "title", "subtitle", "category", "is_active", "duration_ms", "source", "image", "dominant_colours"}).
		AddRow(1, 0, "blah", "", "", false, 0, "", "", models.SerializableColours{})

	p := fakePostgresStore(t, query, rows)
	want := models.ComboDBMediaItem{
		ID:    "1",
		Title: "blah",
	}
	got, err := p.GetNewest()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func fakePostgresStore(t *testing.T, query string, rows *sqlmock.Rows) Store {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	mock.ExpectQuery(query).WillReturnRows(rows)
	return &PostgresStore{
		DB: sqlx.NewDb(db, "sqlmock"),
	}
}
