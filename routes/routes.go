package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/marcus-crane/gunslinger/handlers"
)

func New() *fiber.App {
	app := fiber.New(fiber.Config{
		ServerHeader: "Gunslinger/1.0",
		GETOnly:      true,
	})
	app.Use(cors.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello!")
	})

	api := app.Group("/api")
	v1 := api.Group("/v1", func(c *fiber.Ctx) error {
		c.JSON(fiber.Map{
			"message": "Welcome to v1",
		})
		return c.Next()
	})

	v1.Get("/audio", handlers.GetAudioPlaybackState)
  v1.Get("/media", handlers.GetMediaPlaybackState)

	return app
}
