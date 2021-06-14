package handlers

import (
	"log"
	"os"

	"github.com/Henry-Sarabia/igdb/v2"
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/models"
)

type CurrentGame struct {
	Title string `json:"title"`
	Cover string `json:"image"`
	URL   string `json:"url"`
}

type GameTitle struct {
	Title string `json:"title"`
}

var currentGame CurrentGame

func UpdateGameInFocus(c *fiber.Ctx) error {
	if c.Query("token") == "" {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "You forgot to provide your token",
		})
	}
	if os.Getenv("AUTH_TOKEN") != c.Query("token") {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "Get outta here!",
		})
	}

	videogame := new(GameTitle)

	if err := c.BodyParser(videogame); err != nil {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "Hmm, there was an error with your payload.",
		})
	}

	if videogame.Title == "" {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "You forgot to include the title of the game you're playing",
		})
	}

	clientId := os.Getenv("IGDB_CLIENT_ID")
	accessToken := os.Getenv("IGDB_ACCESS_TOKEN")

	client := igdb.NewClient(clientId, accessToken, nil)

	games, err := client.Games.Search(
		videogame.Title,
		igdb.SetFields("name", "url"),
		igdb.SetLimit(1),
	)

	if err != nil {
		log.Panic(err)
	}

	currentGame = CurrentGame{
		Title: games[0].Name,
		URL:   games[0].URL,
	}

	return c.JSON(models.ResponseHTTP{
		Success: true,
		Data:    currentGame,
	})
}

func ClearGameInFocus(c *fiber.Ctx) error {
	if c.Query("token") == "" {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "You forgot to provide your token",
		})
	}
	if os.Getenv("AUTH_TOKEN") != c.Query("token") {
		return c.JSON(models.ResponseHTTP{
			Success: false,
			Data:    "Get outta here!",
		})
	}
	currentGame = CurrentGame{}
	return c.JSON(models.ResponseHTTP{
		Success: true,
		Data:    "Cleared game",
	})
}

func GetGameInFocus(c *fiber.Ctx) error {
	return c.JSON(models.ResponseHTTP{
		Success: true,
		Data:    currentGame,
	})
}
