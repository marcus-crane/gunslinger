package jobs

import (
	"context"
	"os"

	"github.com/marcus-crane/go-psn-api-fork"

	"github.com/marcus-crane/gunslinger/models"
)

var SonyRefreshToken = ""

func GetPlaystationPresence() {
	ctx := context.Background()
	lang := "en"
	region := "nz"
	npsso := os.Getenv("NPSSO")
	psnApi, err := psn.NewPsnApi(
		lang,
		region,
	)
	if err != nil {
		panic(err)
	}

	if SonyRefreshToken == "" {
		err = psnApi.AuthWithNPSSO(ctx, npsso)
		if err != nil {
			panic(err)
		}
		SonyRefreshToken, _ = psnApi.GetRefreshToken()
	} else {
		err = psnApi.AuthWithRefreshToken(ctx, SonyRefreshToken)
	}

	presence, err := psnApi.GetPresenceRequest(ctx)
	if err != nil {
		panic(err)
	}

	if len(presence.GameTitleInfoList) == 0 {
		if CurrentPlaybackItem.Source == "playstation" {
			CurrentPlaybackItem.IsActive = false
		}
		return
	}

	_, dominantColours := extractImageContent(presence.GameTitleInfoList[0].TitleImage)

	playingItem := models.MediaItem{
		Title:           presence.GameTitleInfoList[0].Name,
		Subtitle:        presence.PrimaryPlatformInfo.Platform,
		Category:        "gaming",
		Source:          "playstation",
		Image:           presence.GameTitleInfoList[0].TitleImage,
		IsActive:        true,
		DominantColours: dominantColours,
	}

	CurrentPlaybackItem = playingItem
}
