package handlers

import (
	"fmt"
	"log"
	"os"

	"github.com/Henry-Sarabia/igdb/v2"
	"github.com/gofiber/fiber/v2"

	"github.com/marcus-crane/gunslinger/models"
)

type CurrentGame struct {
	Title             string    `json:"title"`
	Cover             GameCover `json:"cover"`
	URL               string    `json:"url"`
	Storyline         string    `json:"storyline"`
	Summary           string    `json:"summary"`
	Genres            []int     `json:"genres"`
	InvolvedCompanies []int     `json:"involved_companies"`
}

type GameCover struct {
	ImageURL string `json:"image_url"`
	Height   int    `json:"height"`
	Width    int    `json:"width"`
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
		igdb.SetFields("name", "url", "cover", "genres", "involved_companies", "storyline", "summary"),
		igdb.SetLimit(1),
	)

	if err != nil {
		log.Panic(err)
	}

	cover, err := client.Covers.Get(
		games[0].Cover,
		igdb.SetFields("height", "width", "image_id"),
	)

	gameCover := GameCover{
		ImageURL: fmt.Sprintf(
			"https://images.igdb.com/igdb/image/upload/t_cover_big_2x/%s.jpg",
			cover.ImageID,
		),
		Height: cover.Height,
		Width:  cover.Width,
	}

	currentGame = CurrentGame{
		Title:     games[0].Name,
		Cover:     gameCover,
		URL:       games[0].URL,
		Storyline: games[0].Storyline,
		Summary:   games[0].Summary,
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
