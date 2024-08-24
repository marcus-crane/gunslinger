package main

import (
	"log/slog"
	"net/http"
	"os"

	glc "github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"github.com/marcus-crane/gunslinger/config"
	gdb "github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/migrations"
	"github.com/marcus-crane/gunslinger/playback"
)

func main() {
	cfg := config.Config{}

	dotEnvFeeder := feeder.DotEnv{Path: ".env"}
	envFeeder := feeder.Env{}

	err := glc.
		New().
		AddFeeder(dotEnvFeeder, envFeeder).
		AddStruct(&cfg).
		Feed()

	if err != nil {
		panic(err)
	}

	logLevel := cfg.GetLogLevel()
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(h))
	slog.With(slog.String("log_level", logLevel.Level().String())).Debug("Initialised logger")

	db, err := sqlx.Connect("sqlite", cfg.Gunslinger.DbPath)
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

	jobScheduler := SetupInBackground(cfg, ps, &store)

	if cfg.Gunslinger.BackgroundJobsEnabled {
		jobScheduler.StartAsync()
		slog.Debug("Background jobs have started up in the background.")
	} else {
		slog.Debug("Background jobs are disabled.")
	}

	events.Init()

	router := RegisterRoutes(http.NewServeMux(), cfg, ps)

	slog.Info("Gunslinger is running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("", slog.String("stack", err.Error()))
		jobScheduler.Stop()
		os.Exit(1)
	}
}
