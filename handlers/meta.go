package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/models"
)

func GetIndex(c *fiber.Ctx) error {
	return c.JSON(models.ResponseHTTP{
		Success: true,
	})
}

func GetAPIRoot(c *fiber.Ctx) error {
	return c.JSON(models.ResponseHTTP{
		Success: true,
	})
}

func GetV1Root(c *fiber.Ctx) error {
	return c.JSON(models.ResponseHTTP{
		Success: true,
	})
}