package anilist

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	anilistGraphqlEndpoint = "https://graphql.anilist.co"
)

type AnilistResponse struct {
	Data AnilistData `json:"data"`
}

type AnilistData struct {
	Page Page `json:"Page"`
}

type Page struct {
	Activities []Activity `json:"activities"`
}

type Activity struct {
	Id        int64  `json:"id"`
	Status    string `json:"status"`
	Progress  string `json:"progress"`
	CreatedAt int64  `json:"createdAt"`
	Media     Manga  `json:"media"`
}

type Manga struct {
	Id         int64      `json:"id"`
	Title      MangaTitle `json:"title"`
	Chapters   int        `json:"chapters"`
	CoverImage MangaCover `json:"coverImage"`
}

type MangaTitle struct {
	UserPreferred string `json:"userPreferred"`
}

type MangaCover struct {
	ExtraLarge string `json:"extraLarge"`
}

func GetRecentlyReadManga(cfg config.Config, ps *playback.PlaybackSystem, client http.Client) {
	payload := strings.NewReader("{\"query\":\"query Test {\\n  Page(page: 1, perPage: 10) {\\n    activities(\\n\\t\\t\\tuserId: 6111545\\n      type: MANGA_LIST\\n      sort: ID_DESC\\n    ) {\\n      ... on ListActivity {\\n        id\\n        status\\n\\t\\t\\t\\tprogress\\n        createdAt\\n        media {\\n          chapters\\n          id\\n          title {\\n            userPreferred\\n          }\\n          coverImage {\\n            extraLarge\\n          }\\n        }\\n      }\\n    }\\n  }\\n}\\n\",\"variables\":{}}")
	req, err := http.NewRequest("POST", anilistGraphqlEndpoint, payload)
	if err != nil {
		slog.Error("Failed to build Anilist manga payload", slog.String("error", err.Error()))
		return
	}
	req.Header = http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Bearer %s", cfg.Anilist.Token)},
		"Content-Type":  []string{"application/json"},
		"User-Agent":    []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to contact Anilist for manga updates", slog.String("error", err.Error()))
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read Anilist response", slog.String("error", err.Error()))
		return
	}
	var anilistResponse AnilistResponse

	if err = json.Unmarshal(body, &anilistResponse); err != nil {
		slog.Error("Error fetching Anilist data", slog.String("error", err.Error()))
		return
	}

	if len(anilistResponse.Data.Page.Activities) == 0 {
		slog.Warn("Found no activities for Anilist")
		return
	}

	for _, activity := range anilistResponse.Data.Page.Activities {
		update := playback.Update{
			MediaItem: playback.MediaItem{
				Title:    activity.Progress,
				Subtitle: activity.Media.Title.UserPreferred,
				Category: string(playback.Manga),
				Source:   string(playback.Anilist),
			},
			Status: playback.StatusStopped,
		}

		mediaID := playback.GenerateMediaID(&update)
		exists, err := ps.HasPlaybackEntry(mediaID)
		if err != nil {
			slog.Error("Failed to check Anilist entry existence", slog.String("error", err.Error()))
			continue
		}
		if exists {
			continue
		}

		coverUrl, domColours, err := ps.ResolveCover(cfg, mediaID, activity.Media.CoverImage.ExtraLarge)
		if err != nil {
			slog.Error("Failed to resolve cover for Anilist",
				slog.String("error", err.Error()),
				slog.String("title", update.MediaItem.Title),
			)
			continue
		}
		update.MediaItem.Image = coverUrl
		update.MediaItem.DominantColours = domColours

		if err := ps.UpdatePlaybackState(update); err != nil {
			slog.Error("Failed to save Anilist update",
				slog.String("error", err.Error()),
				slog.String("title", update.MediaItem.Title))
		}
	}
}
