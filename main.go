package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
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

	dsn := utils.MustEnv("DB_PATH")

	database, err := db.NewSqliteStore(dsn)
	if err != nil {
		slog.Error("Failed to create connection to DB", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	if err := database.ApplyMigrations(embedMigrations); err != nil {
		slog.Error("Failed to apply migrations to DB", slog.String("stack", err.Error()))
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
