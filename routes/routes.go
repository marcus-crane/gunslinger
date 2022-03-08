package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/handlers"
	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/models"
)

func Register(app *fiber.App) *fiber.App {
	api := app.Group("/api")
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(models.ResponseHTTP{
			Success: true,
			Data:    "This is the root of Gunslinger's various APIs",
		})
	})

	v1 := api.Group("/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(models.ResponseHTTP{
			Success: true,
			Data:    "This is the v1 endpoint of the API",
		})
	})

	v1.Get("/videogames", handlers.GetGameInFocus)
	v1.Post("/videogames", handlers.UpdateGameInFocus)
	v1.Delete("/videogames", handlers.ClearGameInFocus)

	v2 := api.Group("/v2")
	v2.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(models.ResponseHTTP{
			Success: true,
			Data:    "This is the v2 endpoint of the API. There are no v2 endpoints at present.",
		})
	})

	v3 := api.Group("/v3")
	v3.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(models.ResponseHTTP{
			Success: true,
			Data:    "This is the v3 endpoint of the API",
		})
	})

	v3.Get("/playing", func(c *fiber.Ctx) error {
		return c.JSON(jobs.CurrentPlaybackItem)
	})

	return app
}
