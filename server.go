package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	// "github.com/marcus-crane/gunslinger/database"
	"github.com/marcus-crane/gunslinger/jobs"
	// "github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/routes"
)

func main() {
	// if err := database.Connect(); err != nil {
	// 	log.Panic("Can't connect to database:", err.Error())
	// }

	//database.DBConn.AutoMigrate(&models.Audio{})

	jobs.BackgroundSetup()

	app := routes.New()

	go func() {
    fmt.Println("Listening at http://localhost:8080")
		if err := app.Listen(":8080"); err != nil {
			log.Panic(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	_ = <-c
	fmt.Println("Gracefully shutting down...")
	_ = app.Shutdown()

	fmt.Println("Running cleanup tasks")

	// Shutdown task here

	fmt.Println("gunslinger has successfully shut down.")
}
