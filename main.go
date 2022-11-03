package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/routes"
)

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
	}

	database := db.Initialize()

	jobScheduler := jobs.SetupInBackground(database)

	if os.Getenv("BACKGROUND_JOBS_ENABLED") == "true" {
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
