package migrations

import (
	"embed"
)

//go:embed *.sql
var embedMigrations embed.FS

func GetMigrations() embed.FS {
	return embedMigrations
}
