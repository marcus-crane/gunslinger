package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/routes"
	"github.com/marcus-crane/gunslinger/utils"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
	}

	dbName := "sqlite"

	dsn := utils.MustEnv("DB_PATH")

	database := db.Initialize(dbName, dsn)

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal().Err(err).Msg("Failed to set sqlte3 dialect")
	}

	if err := goose.Up(database.DB, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("Failed to run goose migration")
	}

	jobScheduler := jobs.SetupInBackground(database)

	if utils.GetEnv("BACKGROUND_JOBS_ENABLED", "true") == "true" {
		jobScheduler.StartAsync()
		log.Info().Msg("Background jobs have started up in the background.")
	} else {
		log.Info().Msg("Background jobs are disabled.")
	}

	events.Init()

	router := routes.Register(http.NewServeMux(), database)

	log.Info().Msg("Gunslinger is running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Error().Err(err).Send()
		jobScheduler.Stop()
		os.Exit(1)
	}
}
