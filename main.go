package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

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

	database := db.Initialize()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("Failed to set sqlite3 dialect: %+v\n", err)
	}

	if err := goose.Up(database.DB, "migrations"); err != nil {
		log.Fatalf("Failed to run goose migration: %+v\n", err)
	}

	jobScheduler := jobs.SetupInBackground(database)

	if utils.GetEnv("BACKGROUND_JOBS_ENABLED", "true") == "true" {
		jobScheduler.StartAsync()
		fmt.Println("Background jobs have started up in the background.")
	} else {
		fmt.Println("Background jobs are disabled.")
	}

	events.Init()

	router := routes.Register(http.NewServeMux(), database)

	fmt.Println("Gunslinger is running at http://localhost:8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		fmt.Println(err)
		jobScheduler.Stop()
		os.Exit(1)
	}
}
