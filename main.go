package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"golang.org/x/exp/slog"

	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/utils"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
	}

	dsn := utils.MustEnv("DB_PATH")

	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		slog.Error("Failed to create connection to DB", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	ps := NewPlaybackSystem(db)

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect(string(goose.DialectSQLite3)); err != nil {
		slog.Error("Failed to set goose dialect", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	if err := goose.Up(db.DB, "migrations"); err != nil {
		slog.Error("Failed to create connection to DB", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	jobScheduler := SetupInBackground(ps)

	if utils.GetEnv("BACKGROUND_JOBS_ENABLED", "true") == "true" {
		jobScheduler.StartAsync()
		slog.Info("Background jobs have started up in the background.")
	} else {
		slog.Info("Background jobs are disabled.")
	}

	events.Init()

	router := RegisterRoutes(http.NewServeMux(), ps)

	slog.Info("Gunslinger is running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("", slog.String("stack", err.Error()))
		jobScheduler.Stop()
		os.Exit(1)
	}
}
