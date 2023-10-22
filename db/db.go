package db

import (
	"embed"

	"github.com/jmoiron/sqlx"

	"github.com/marcus-crane/gunslinger/models"

	_ "modernc.org/sqlite"
)

func Initialize(dbName string, dsn string) *sqlx.DB {
	return sqlx.MustConnect(dbName, dsn)
}

type Store interface {
	ApplyMigrations(migrations embed.FS) error
	GetRecent() ([]models.ComboDBMediaItem, error)
	GetNewest() (models.ComboDBMediaItem, error)
	GetByCategory(category string) (models.ComboDBMediaItem, error)
	Insert(item models.MediaItem) error
	GetCustom(query string, args ...interface{}) (models.ComboDBMediaItem, error)
	ExecCustom(query string, args ...interface{}) error
}
