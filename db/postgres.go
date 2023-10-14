package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/models"
)

type PostgresStore struct {
	DB *sqlx.DB
}

func (p *PostgresStore) GetConnection() *sqlx.DB {
	return p.DB
}

func (p *PostgresStore) RetrieveRecent() ([]models.ComboDBMediaItem, error) {
	cl := []models.ComboDBMediaItem{}
	if err := p.DB.Select(&cl, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 7"); err != nil {
		return cl, err
	}
	return cl, nil
}

func (p *PostgresStore) RetrieveLatest() (models.ComboDBMediaItem, error) {
	c := models.ComboDBMediaItem{}
	row := p.DB.QueryRowx("SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 1")
	err := row.StructScan(&c)
	if err != nil {
		return c, err
	}
	return c, nil
}
