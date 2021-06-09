package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/models"
)

func GetAudioPlaybackState(c *fiber.Ctx) error {
	return c.JSON(models.ResponseHTTP{
		Success: true,
		Data:    jobs.AudioPlaybackStatus,
	})
}
