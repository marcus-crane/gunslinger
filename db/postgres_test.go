package db

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/models"
)

func TestPostgresStore_RetrieveAll(t *testing.T) {
	t.Parallel()
	p := fakePostgresStore(t)
	want := []models.ComboDBMediaItem{
		{
			ID:    1,
			Title: "blah",
		},
		{
			ID:    2,
			Title: "bleh",
		},
	}
	got, err := p.RetrieveAll()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func fakePostgresStore(t *testing.T) PostgresStore {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	query := "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 7"
	rows := sqlmock.NewRows([]string{"id", "created_at", "title", "subtitle", "category", "is_active", "duration_ms", "source", "image", "dominant_colours"}).
		AddRow(1, 0, "blah", "", "", false, 0, "", "", models.SerializableColours{}).
		AddRow(2, 0, "bleh", "", "", false, 0, "", "", models.SerializableColours{})
	mock.ExpectQuery(query).WillReturnRows(rows)
	return PostgresStore{
		DB: sqlx.NewDb(db, "sqlmock"),
	}
}
