package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"golang.org/x/exp/slog"

	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/routes"
	"github.com/marcus-crane/gunslinger/utils"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
	}

	dbName := "sqlite"

	dsn := utils.MustEnv("DB_PATH")

	database := db.Initialize(dbName, dsn)

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		slog.Error("Failed to set sqlite3 dialect", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	if err := goose.Up(database.DB, "migrations"); err != nil {
		slog.Error("Failed to run goose migration", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	jobScheduler := jobs.SetupInBackground(database)

	if utils.GetEnv("BACKGROUND_JOBS_ENABLED", "true") == "true" {
		jobScheduler.StartAsync()
		slog.Info("Background jobs have started up in the background.")
	} else {
		slog.Info("Background jobs are disabled.")
	}

	events.Init()

	router := routes.Register(http.NewServeMux(), database)

	slog.Info("Gunslinger is running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("", slog.String("stack", err.Error()))
		jobScheduler.Stop()
		os.Exit(1)
	}
}
