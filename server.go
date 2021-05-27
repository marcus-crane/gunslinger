package main

import (
  "fmt"
  "log"
  "os"
  "os/signal"
  "syscall"

  "github.com/marcus-crane/swissarmy/database"
  "github.com/marcus-crane/swissarmy/models"
  "github.com/marcus-crane/swissarmy/routes"
)

func main() {
  if err := database.Connect(); err != nil {
    log.Panic("Can't connect to database:", err.Error())
  }

  database.DBConn.AutoMigrate(&models.Song{})

  app := routes.New()

  go func() {
    if err := app.Listen(":3000"); err != nil {
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

  fmt.Println("swissarmy has successfully shut down.")
}
