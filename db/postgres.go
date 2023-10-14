package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/models"
)

type PostgresStore struct {
	DB *sqlx.DB
}

func (p *PostgresStore) Store(c models.ComboDBMediaItem) (uint, error) {
	return 1, nil
}

func (p *PostgresStore) RetrieveAll() ([]models.ComboDBMediaItem, error) {
	cl := []models.ComboDBMediaItem{}
	if err := p.DB.Select(&cl, "SELECT id, created_at, title, subtitle, category, is_active, duration_ms, source, image, dominant_colours FROM db_media_items ORDER BY created_at desc LIMIT 7"); err != nil {
		return cl, err
	}
	return cl, nil
}
