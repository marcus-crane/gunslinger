package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/models"
)

func GetIndex(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Title": "Howdy pardner",
	})
}

func GetAPIRoot(c *fiber.Ctx) error {
	return c.JSON(models.ResponseHTTP{
		Success: true,
		Data:    "This is the base of the API",
	})
}

func GetV1Root(c *fiber.Ctx) error {
	return c.JSON(models.ResponseHTTP{
		Success: true,
		Data:    "This is the v1 endpoint of the API",
	})
}
