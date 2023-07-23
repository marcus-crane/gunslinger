package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/utils"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
)

func Initialize() *sqlx.DB {
	db, err := sqlx.Connect("sqlite", utils.MustEnv("DB_PATH"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to DB")
	}
	log.Info().Msg("Successfully connected to DB")
	return db
}
