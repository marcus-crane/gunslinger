package db

import (
	"embed"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"github.com/marcus-crane/gunslinger/models"

	_ "modernc.org/sqlite"
)

type SqliteStore struct {
	DB *sqlx.DB
}

func NewSqliteStore(dsn string) (Store, error) {
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &SqliteStore{
		DB: db,
	}, nil
}

func (s *SqliteStore) GetConnection() *sqlx.DB {
	return s.DB
}

func (s *SqliteStore) ApplyMigrations(migrations embed.FS) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect(string(goose.DialectSQLite3)); err != nil {
		return err
	}

	if err := goose.Up(s.DB.DB, "migrations"); err != nil {
		return err
	}

	return nil
}

func (s *SqliteStore) GetRecent() ([]models.ComboDBMediaItem, error) {
	cl := []models.ComboDBMediaItem{}
	if err := s.DB.Select(&cl, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 7"); err != nil {
		return cl, err
	}
	return cl, nil
}

func (s *SqliteStore) GetNewest() (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get(&c, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 1")
	if err != nil {
		return c, err
	}
	return c, nil
}

func (s *SqliteStore) GetByCategory(category string) (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get("SELECT * FROM db_media_items WHERE category = ? ORDER BY created_at desc LIMIT 1", category)
	if err != nil {
		return c, err
	}
	return c, nil
}
