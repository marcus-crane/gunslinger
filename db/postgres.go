package db

import (
	"embed"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"github.com/marcus-crane/gunslinger/models"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	DB *sqlx.DB
}

func NewPostgresStore(dsn string) (Store, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresStore{
		DB: db,
	}, nil
}

func (s *PostgresStore) GetConnection() *sqlx.DB {
	return s.DB
}

func (s *PostgresStore) ApplyMigrations(migrations embed.FS) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect(string(goose.DialectPostgres)); err != nil {
		return err
	}

	if err := goose.Up(s.DB.DB, "migrations"); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) GetRecent() ([]models.ComboDBMediaItem, error) {
	cl := []models.ComboDBMediaItem{}
	if err := s.DB.Select(&cl, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 7"); err != nil {
		return cl, err
	}
	return cl, nil
}

func (s *PostgresStore) GetNewest() (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get(&c, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 1")
	if err != nil {
		return c, err
	}
	return c, nil
}

func (s *PostgresStore) GetByCategory(category string) (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get("SELECT * FROM db_media_items WHERE category = ? ORDER BY created_at desc LIMIT 1", category)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (s *PostgresStore) Insert(item models.MediaItem) error {
	_, err := s.DB.Exec(
		"INSERT INTO db_media_items (created_at, title, subtitle, category, is_active, duration_ms, dominant_colours, source, image) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		time.Now().Unix(),
		item.Title,
		item.Subtitle,
		item.Category,
		item.IsActive,
		item.Duration,
		item.DominantColours,
		item.Source,
		item.Image,
	)
	return err
}
