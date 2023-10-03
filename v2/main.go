package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/marcus-crane/gunslinger/v2/plex"
	"github.com/marcus-crane/gunslinger/v2/steam"
	"github.com/marcus-crane/gunslinger/v2/utils"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
	}

	steamApiKey := utils.MustEnv("STEAM_API_KEY")

	steam := steam.NewClient(steamApiKey)
	media, err := steam.QueryMediaState()
	if err != nil {
		fmt.Printf("Got no result for Steam: %+v\n", err)
	}
	fmt.Printf("Steam result: %+v\n", media)

	plexApiKey := utils.MustEnv("PLEX_API_KEY")

	plex := plex.NewClient(plexApiKey)
	media2, err := plex.QueryMediaState()
	if err != nil {
		fmt.Printf("Got no result for Plex: %+v\n", err)
	}
	for _, im := range media2 {
		fmt.Printf("%s - %s ( %s )", im.Title, im.Subtitle, im.Author)
	}
}
