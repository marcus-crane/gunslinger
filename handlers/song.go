package handlers

import (
  "net/http"

  "github.com/gofiber/fiber/v2"

  "github.com/marcus-crane/gunslinger/database"
  "github.com/marcus-crane/gunslinger/models"
)

type ResponseHTTP struct {
  Success bool        `json:"success"`
  Data    interface{} `json:"data"`
  Message string      `json:"message"`
}

func GetAllSongs(c *fiber.Ctx) error {
  db := database.DBConn

  var songs []models.Song
  if res := db.Find(&songs); res.Error != nil {
    return c.Status(http.StatusServiceUnavailable).JSON(ResponseHTTP{
      Success: false,
      Message: res.Error.Error(),
      Data:    nil,
    })
  }

  return c.JSON(ResponseHTTP{
    Success: true,
    Message: "Successfully retrieved all songs",
    Data:    songs,
  })
}
