package jobs

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/r3labs/sse/v2"
)

const (
	anilistGraphqlEndpoint = "https://graphql.anilist.co"
)

func GetRecentlyReadManga(database *sqlx.DB) {
	var client http.Client
	payload := strings.NewReader("{\"query\":\"query Test {\\n  Page(page: 1, perPage: 10) {\\n    activities(\\n\\t\\t\\tuserId: 6111545\\n      type: MANGA_LIST\\n      sort: ID_DESC\\n    ) {\\n      ... on ListActivity {\\n        id\\n        status\\n\\t\\t\\t\\tprogress\\n        createdAt\\n        media {\\n          id\\n          title {\\n            userPreferred\\n          }\\n          coverImage {\\n            extraLarge\\n          }\\n        }\\n      }\\n    }\\n  }\\n}\\n\",\"variables\":{}}")
	req, err := http.NewRequest("POST", anilistGraphqlEndpoint, payload)
	if err != nil {
		panic(err)
	}
	req.Header = http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Bearer %s", os.Getenv("ANILIST_TOKEN"))},
		"Content-Type":  []string{"application/json"},
		"User-Agent":    []string{UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var anilistResponse models.AnilistResponse

	if err = json.Unmarshal(body, &anilistResponse); err != nil {
		fmt.Println("Error fetching Anilist data: ", err)
		return
	}

	updateOccured := false

	for _, activity := range anilistResponse.Data.Page.Activities {
		if activity.Status == "read chapter" {
			var existingItem models.DBMediaItem
			// Have we saved this update already?
			if err := database.Get(
				&existingItem,
				"SELECT * FROM db_media_items WHERE category = ? AND title = ? ORDER BY created_at desc LIMIT 1",
				"manga",
				activity.Progress,
			); err == nil {
				continue // We have already seen this status update
			}

			// Has this status update changed? If so, it will start with the same start chapter
			if strings.Contains(activity.Progress, " - ") {
				startChapter := strings.Split(activity.Progress, " - ")[0]

				// First, we check if the existing chapter is also part of a range ie; 102 - 105
				if err := database.Get(
					&existingItem,
					"SELECT * FROM db_media_items WHERE category = ? AND title LIKE ? ORDER BY created_at desc LIMIT 1",
					"manga",
					"%"+fmt.Sprintf("%s - ", startChapter)+"%", // Make sure we include - so eg; Chapter 100 doesn't partial match Chapter 1000
				); err == nil {
					// Found an existing update so we need to update the end chapter
					err := updateChapter(database, activity, existingItem)
					if err != nil {
						fmt.Printf("Failed to update entry for %d %s", activity.Id, activity.Media.Title.UserPreferred)
					} else {
						fmt.Println("Saved")
						updateOccured = true
					}
					continue
				}

				// Next, we check if the existing chapter was the first in a streak ie; 102
				if err := database.Get(
					&existingItem,
					"SELECT * FROM db_media_items WHERE category = ? AND title = ? ORDER BY created_at desc LIMIT 1",
					"manga",
					startChapter,
				); err == nil {
					err := updateChapter(database, activity, existingItem)
					if err != nil {
						fmt.Printf("Failed to update entry for %d %s", activity.Id, activity.Media.Title.UserPreferred)
					} else {
						fmt.Println("Saved")
						updateOccured = true
					}
					continue
				}
			}

			image, extension, _ := extractImageContent(activity.Media.CoverImage.ExtraLarge)

			imageHash := md5.Sum(image)
			var genericBytes []byte = imageHash[:] // Disgusting :)
			guid, _ := uuid.FromBytes(genericBytes)
			discImage := fmt.Sprintf("/static/cover.%s.%s", guid, extension)

			if err := saveCover(guid.String(), image, extension); err != nil {
				fmt.Printf("Failed to save cover for Anilist: %+v\n", err)
			}

			// We haven't seen this chapter update so we'll save it
			schema := `INSERT INTO db_media_items (created_at, title, subtitle, category, is_active, source, image) VALUES (?, ?, ?, ?, ?, ?, ?)`
			_, err := database.Exec(
				schema,
				activity.CreatedAt,
				activity.Progress,
				activity.Media.Title.UserPreferred,
				"manga",
				false,
				"anilist",
				discImage,
			)
			if err != nil {
				fmt.Printf("Failed to save DB entry for manga: %+v\n", err)
				continue
			}
			updateOccured = true
		}
	}

	if updateOccured {
		var latestItem models.DBMediaItem
		if err := database.Get(
			&latestItem,
			"SELECT * FROM db_media_items WHERE category = ? ORDER BY created_at desc LIMIT 1",
			"manga",
		); err == nil {
			// If we've read manga in the past but only just fetched updates, we don't consider this "live"
			// so only update the live player if nothing else is live and manga is more recent
			if !CurrentPlaybackItem.IsActive && latestItem.CreatedAt > CurrentPlaybackItem.CreatedAt {
				playingItem := models.MediaItem{
					CreatedAt: latestItem.CreatedAt,
					Title:     latestItem.Title,
					Subtitle:  latestItem.Subtitle,
					Category:  latestItem.Category,
					Source:    latestItem.Source,
					IsActive:  latestItem.IsActive,
					Image:     latestItem.Image,
				}
				byteStream := new(bytes.Buffer)
				json.NewEncoder(byteStream).Encode(playingItem)
				events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
				CurrentPlaybackItem = playingItem
			}
		}
	}
}

func updateChapter(database *sqlx.DB, activity models.Activity, existingItem models.DBMediaItem) error {
	query := `UPDATE db_media_items SET created_at = ?, title = ? WHERE id = ?`
	_, err := database.Exec(
		query,
		activity.CreatedAt,
		activity.Progress,
		existingItem.ID,
	)
	return err
}
