package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"

	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/routes"
)

func main() {

	err := godotenv.Load()

	if err != nil {
		log.Print("Couldn't find .env file. Continuing on from environment anyway.")
	}

	jobScheduler := jobs.SetupInBackground()

	if os.Getenv("BACKGROUND_JOBS_ENABLED") == "true" {
		jobScheduler.StartAsync()
		log.Print("Background jobs have started up in the background.")
	} else {
		log.Print("Background jobs are disabled.")
	}

	engine := html.New("./views", ".html")

	if os.Getenv("DEBUG") == "true" {
		engine.Debug(true)
		log.Print("Running fiber in debug mode")
	}

	if os.Getenv("DEVELOPMENT") == "true" {
		engine.Reload(true)
		log.Print("Running fiber in development mode")
	}

	app := fiber.New(fiber.Config{
		Views:        engine,
		ServerHeader: "Gunslinger/1.0",
		GETOnly:      true,
	})

	app.Static("/", "./static")

	app.Use(func (c *fiber.Ctx) error {
		log.Print(c.Protocol())
		return c.Next()
	})

	app.Use(logger.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://utf9k.net",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app = routes.Register(app)

	go func() {
		log.Print("Listening at http://localhost:8080")
		if err := app.Listen(":8080"); err != nil {
			log.Panic(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	_ = <-c
	log.Print("Gracefully shutting down...")
	_ = app.Shutdown()

	log.Print("Running cleanup tasks")

	jobScheduler.Stop()

	log.Println("Stopping job scheduler")

	log.Print("gunslinger has successfully shut down.")
}
