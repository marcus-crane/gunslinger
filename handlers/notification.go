package handlers

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gregdel/pushover"

	"github.com/marcus-crane/gunslinger/models"
)

type SiteImpression struct {
	LikedURL string `json:"liked_url"`
}

func NotifyPositiveSiteImpression(c *fiber.Ctx) error {
	impression := new(SiteImpression)

	if err := c.BodyParser(impression); err != nil {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "Hmm, something went horribly wrong. Perhaps try again later?",
		})
	}

	if impression.LikedURL == "" {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "You forgot to include a URL?! This seems buggy... or fishy?",
		})
	}

	if !strings.Contains(impression.LikedURL, "utf9k.net") {
		return c.JSON(models.ResponseHTTP{
			Success: true,
			Data:    "Nice try but you shouldn't be thanking me for other people's websites.",
		})
	}

	app := pushover.New(os.Getenv("PUSHOVER_APP_TOKEN"))
	recipient := pushover.NewRecipient(os.Getenv("PUSHOVER_USER_ID"))
	message := &pushover.Message{
		Message:    fmt.Sprintf("They appreciated the page located at %s", impression.LikedURL),
		Title:      "A utf9k reader passes on their thanks!",
		URL:        impression.LikedURL,
		URLTitle:   "utf9k",
		DeviceName: "iPhone12Pro",
	}
	_, err := app.SendMessage(message, recipient)
	if err != nil {
		log.Panic(err)
	}
	return c.JSON(models.ResponseHTTP{
		Success: true,
		Data:    "I received your thanks anonymously! Much appreciated.",
	})
}