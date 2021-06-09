package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/handlers"
)

func Register(app *fiber.App) *fiber.App {
	app.Get("/", handlers.GetIndex)
	api := app.Group("/api", handlers.GetAPIRoot)
	v1 := api.Group("/v1", handlers.GetV1Root)

	v1.Get("/audio", handlers.GetAudioPlaybackState)
  v1.Get("/media", handlers.GetMediaPlaybackState)

	return app
}
