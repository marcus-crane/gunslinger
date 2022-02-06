package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/handlers"
)

func Register(app *fiber.App) *fiber.App {
	app.Get("/", handlers.GetIndex)

	api := app.Group("/api")
	api.Get("/", handlers.GetAPIRoot)

	v1 := api.Group("/v1")
	v1.Get("/", handlers.GetV1Root)

	v1.Get("/videogames", handlers.GetGameInFocus)
	v1.Post("/videogames", handlers.UpdateGameInFocus)
	v1.Delete("/videogames", handlers.ClearGameInFocus)
	v1.Post("/thanks", handlers.NotifyPositiveSiteImpression)

	v2 := api.Group("/v2")
	v2.Get("/", handlers.GetV2Root)

	v3 := api.Group("/v3")
	v3.Get("/", handlers.GetV3Root)

	v3.Get("/playing", handlers.GetCurrentlyPlaying)

	return app
}
