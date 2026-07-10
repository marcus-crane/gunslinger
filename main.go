package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	glc "github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"github.com/marcus-crane/gunslinger/config"
	gdb "github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/migrations"
	"github.com/marcus-crane/gunslinger/playback"
)

func main() {
	cfg := config.Config{}

	envFeeder := feeder.Env{}
	feeders := []glc.Feeder{envFeeder}

	if _, err := os.Stat(".env"); err == nil {
		// We only load in .env if one is present or else we crash with file not exists error
		feeders = append(feeders, feeder.DotEnv{Path: ".env"})
	}

	err := glc.
		New().
		AddFeeder(feeders...).
		AddStruct(&cfg).
		Feed()

	if err != nil {
		panic(err)
	}

	logLevel := cfg.GetLogLevel()
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(h))
	slog.With(slog.String("log_level", logLevel.Level().String())).Debug("Initialised logger")

	// See https://blog.pecar.me/sqlite-prod
	dsn := fmt.Sprintf("file:%s?%s", cfg.Gunslinger.DbPath, strings.Join([]string{
		"_pragma=busy_timeout(5000)",
		"_pragma=journal_mode(WAL)",
		"_pragma=foreign_keys(1)",
		"_pragma=synchronous(NORMAL)",
		"_pragma=mmap_size(134217728)",
		"_pragma=journal_size_limit(27103364)",
		"_pragma=cache_size(2000)",
	}, "&"))
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		slog.Error("Failed to create connection to DB", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ps := playback.NewPlaybackSystem(db)
	store := gdb.SqliteStore{DB: db}

	goose.SetBaseFS(migrations.GetMigrations())

	if err := goose.SetDialect(string(goose.DialectSQLite3)); err != nil {
		slog.Error("Failed to set goose dialect", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := goose.Up(db.DB, "."); err != nil {
		slog.Error("Failed to create connection to DB", slog.String("error", err.Error()))
		os.Exit(1)
	}

	jobScheduler, err := SetupInBackground(cfg, ps, &store)
	if err != nil {
		slog.Error("Failed to start up scheduler", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.Gunslinger.BackgroundJobsEnabled {
		jobScheduler.Start()
		slog.Debug("Background jobs have started up in the background.")
	} else {
		slog.Debug("Background jobs are disabled.")
	}

	router := RegisterRoutes(http.NewServeMux(), cfg, ps, &store)

	slog.Info("Gunslinger is running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("Failed while serving", slog.String("error", err.Error()))
		_ = jobScheduler.Shutdown()
		os.Exit(1)
	}
}
