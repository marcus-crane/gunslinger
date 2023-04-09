package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/utils"
	_ "modernc.org/sqlite"
)

func Initialize() *sqlx.DB {
	db, err := sqlx.Connect("sqlite", utils.MustEnv("DB_PATH"))
	if err != nil {
		log.Fatalf("Failed to connect to DB: %+v\n", err)
	}
	log.Print("Initialised DB connection")
	return db
}
