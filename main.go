package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"golang.org/x/exp/slog"

	gdb "github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/migrations"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/utils"
)

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

	// See https://blog.pecar.me/sqlite-prod
	db.Exec("PRAGMA foreign_keys = ON")
	db.Exec("PRAGMA journal_mode = WAL")
	db.Exec("PRAGMA synchronous = NORMAL")
	db.Exec("PRAGMA mmap_size = 134217728")
	db.Exec("PRAGMA journal_size_limit = 27103364")
	db.Exec("PRAGMA cache_size = 2000")

	ps := playback.NewPlaybackSystem(db)
	store := gdb.SqliteStore{DB: db}

	goose.SetBaseFS(migrations.GetMigrations())

	if err := goose.SetDialect(string(goose.DialectSQLite3)); err != nil {
		slog.Error("Failed to set goose dialect", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	if err := goose.Up(db.DB, "."); err != nil {
		slog.Error("Failed to create connection to DB", slog.String("stack", err.Error()))
		os.Exit(1)
	}

	jobScheduler := SetupInBackground(ps, &store)

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
