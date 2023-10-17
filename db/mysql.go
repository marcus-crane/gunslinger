package db

import (
	"embed"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"github.com/marcus-crane/gunslinger/models"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlStore struct {
	DB *sqlx.DB
}

func NewMysqlStore(dsn string) (Store, error) {
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &MysqlStore{
		DB: db,
	}, nil
}

func (s *MysqlStore) ApplyMigrations(migrations embed.FS) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect(string(goose.DialectMySQL)); err != nil {
		return err
	}

	if err := goose.Up(s.DB.DB, "migrations/mysql"); err != nil {
		return err
	}

	return nil
}

func (s *MysqlStore) GetRecent() ([]models.ComboDBMediaItem, error) {
	cl := []models.ComboDBMediaItem{}
	if err := s.DB.Select(&cl, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 7"); err != nil {
		return cl, err
	}
	return cl, nil
}

func (s *MysqlStore) GetNewest() (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get(&c, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 1")
	if err != nil {
		return c, err
	}
	return c, nil
}

func (s *MysqlStore) GetByCategory(category string) (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get(&c, "SELECT * FROM db_media_items WHERE category = ? ORDER BY created_at desc LIMIT 1", category)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (s *MysqlStore) Insert(item models.MediaItem) error {
	_, err := s.DB.Exec(
		"INSERT INTO db_media_items (created_at, title, subtitle, category, is_active, duration_ms, dominant_colours, source, image) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		item.CreatedAt,
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

func (s *MysqlStore) InsertCustom(query string, args ...interface{}) (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get(&c, query, args...)
	if err != nil {
		return c, err
	}
	return c, nil
}
