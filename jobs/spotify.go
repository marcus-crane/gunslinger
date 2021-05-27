package jobs

import (
  "fmt"

  "github.com/gofiber/fiber/v2"
)

func GetCurrentlyPlaying() {
  client := fiber.AcquireAgent()

  req := client.Request()
  req.SetRequestURI("https://example.com")

  if err := client.Parse(); err != nil {
    panic(err)
  }

  code, body, err := client.String()

  if err != nil {
    panic(err)
  }

  fmt.Println("Got response code")
  fmt.Println(code)

  fmt.Println("Body")
  fmt.Println(body)
}

