package db

import (
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func Initialize(dbName string, dsn string) *sqlx.DB {
	return sqlx.MustConnect(dbName, dsn)
}
