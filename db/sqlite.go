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
	err := s.DB.Get(&c, "SELECT * FROM db_media_items WHERE category = ? ORDER BY created_at desc LIMIT 1", category)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (s *SqliteStore) Insert(item models.MediaItem) error {
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

func (s *SqliteStore) GetTokenByID(id string) string {
	t := models.Token{}
	err := s.DB.Get(&t, "SELECT * FROM tokens WHERE id = ?", id)
	if err != nil {
		return ""
	}
	return t.Value
}

func (s *SqliteStore) GetTokenMetadataByID(id string) models.TokenMetadata {
	t := models.TokenMetadata{}
	err := s.DB.Get(&t, "SELECT * FROM tokenmetadata WHERE id = ?", id)
	if err != nil {
		return models.TokenMetadata{}
	}
	return t
}

func (s *SqliteStore) UpsertToken(id, value string) error {
	query := `
	INSERT INTO tokens (id, value)
	VALUES (?, ?)
	ON CONFLICT (id) DO UPDATE SET
	value = excluded.value
	WHERE id = ?
	`
	_, err := s.DB.Exec(query, id, value, id)
	return err
}

func (s *SqliteStore) UpsertTokenMetadata(id string, createdat, expiresin int64) error {
	query := `
	INSERT INTO tokenmetadata (id, createdat, expiresin)
	VALUES (?, ?, ?)
	ON CONFLICT (id) DO UPDATE SET
	createdat = excluded.createdat,
	expiresin = excluded.expiresin
	WHERE id = ?
	`
	_, err := s.DB.Exec(query, id, createdat, expiresin, id)
	return err
}

func (s *SqliteStore) GetCustom(query string, args ...interface{}) (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	err := s.DB.Get(&c, query, args...)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (s *SqliteStore) ExecCustom(query string, args ...interface{}) error {
	_, err := s.DB.Exec(query, args...)
	return err
}
