package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/jobs"
)

func GetCurrentlyPlaying(c *fiber.Ctx) error {
	return c.JSON(jobs.CurrentPlaybackItem)
}

func GetCurrentProgress(c *fiber.Ctx) error {
	return c.JSON(jobs.CurrentPlaybackProgress)
}