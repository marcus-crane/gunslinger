package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
	"github.com/r3labs/sse/v2"
)

const (
	anilistGraphqlEndpoint = "https://graphql.anilist.co"
)

func GetRecentlyReadManga(database *sqlx.DB, store db.Store, client http.Client) {
	payload := strings.NewReader("{\"query\":\"query Test {\\n  Page(page: 1, perPage: 10) {\\n    activities(\\n\\t\\t\\tuserId: 6111545\\n      type: MANGA_LIST\\n      sort: ID_DESC\\n    ) {\\n      ... on ListActivity {\\n        id\\n        status\\n\\t\\t\\t\\tprogress\\n        createdAt\\n        media {\\n          chapters\\n          id\\n          title {\\n            userPreferred\\n          }\\n          coverImage {\\n            extraLarge\\n          }\\n        }\\n      }\\n    }\\n  }\\n}\\n\",\"variables\":{}}")
	req, err := http.NewRequest("POST", anilistGraphqlEndpoint, payload)
	if err != nil {
		slog.Error("Failed to build Anilist manga payload", slog.String("stack", err.Error()))
		return
	}
	req.Header = http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Bearer %s", os.Getenv("ANILIST_TOKEN"))},
		"Content-Type":  []string{"application/json"},
		"User-Agent":    []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to contact Anilist for manga updates", slog.String("stack", err.Error()))
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read Anilist response", slog.String("stack", err.Error()))
		return
	}
	var anilistResponse models.AnilistResponse

	if err = json.Unmarshal(body, &anilistResponse); err != nil {
		slog.Error("Error fetching Anilist data", slog.String("stack", err.Error()))
		return
	}

	updateOccured := false

	for _, activity := range anilistResponse.Data.Page.Activities {
		if activity.Status == "completed" {
			var previousEntry models.ComboDBMediaItem
			// has it been at least 24 hours since the last update?

			// have we binged it from start to finish somehow?
			if err := database.Get(
				&previousEntry,
				"SELECT * FROM db_media_items WHERE category = ? AND subtitle = ? ORDER BY created_at desc LIMIT 1",
				"manga",
				activity.Media.Title.UserPreferred,
			); err == nil {
				// We have completed this manga entirely but have never surfaced it
				// This would seem quite strange given my current manga set up so
				// for now we'll skip this in case we're backfilling manga
				continue
			}

			// To be able to complete it, it must have chapters in the first place but
			// we should do a sanity check regardless
			lastChapter := activity.Media.Chapters

			if lastChapter == 0 {
				// This should be impossible as a series needs a chapter count
				// to be "completeable" but stranger things have happened.
				// We'll consider this a no-op and bail out
				continue
			}

			streakStartChapter := ""

			if strings.Contains(previousEntry.Title, " - ") {
				streakStartChapter = strings.Split(previousEntry.Title, " - ")[1]
			} else {
				streakStartChapter = previousEntry.Title
			}

			streakStart, err := strconv.Atoi(streakStartChapter)
			if err != nil {
				// Who cares enough really. We'll just skip this event
				continue
			}
			trueStreakStart := streakStart + 1
			// We'll substitute it for a fancy name
			activity.Progress = fmt.Sprintf("Chapters %d - %d (END)", trueStreakStart, activity.Media.Chapters)

			if err != nil {
				slog.Error("Failed to update entry",
					slog.Int64("activity.id", activity.Id),
					slog.String("activity.title", activity.Media.Title.UserPreferred),
					slog.String("stack", err.Error()),
				)
			} else {
				slog.Debug("Updated title successfully", slog.String("activity.title", activity.Media.Title.UserPreferred))
				updateOccured = true
			}
			continue
		}

		if activity.Status == "read chapter" {
			var existingItem models.ComboDBMediaItem
			// Have we saved this update already?
			if err := database.Get(
				&existingItem,
				"SELECT * FROM db_media_items WHERE category = ? AND title = ? AND subtitle = ? ORDER BY created_at desc LIMIT 1",
				"manga",
				activity.Progress,
				activity.Media.Title.UserPreferred,
			); err == nil {
				continue // We have already seen this status update
			}

			// Has this status update changed? If so, it will start with the same start chapter
			if strings.Contains(activity.Progress, " - ") {
				startChapter := strings.Split(activity.Progress, " - ")[0]

				// First, we check if the existing chapter is also part of a range ie; 102 - 105
				if err := database.Get(
					&existingItem,
					"SELECT * FROM db_media_items WHERE category = ? AND title LIKE ? AND subtitle = ? ORDER BY created_at desc LIMIT 1",
					"manga",
					"%"+fmt.Sprintf("%s - ", startChapter)+"%", // Make sure we include - so eg; Chapter 100 doesn't partial match Chapter 1000
					activity.Media.Title.UserPreferred,
				); err == nil {
					// Found an existing update so we need to update the end chapter
					err := updateChapter(database, activity, existingItem)
					if err != nil {
						slog.Error("Failed to update entry",
							slog.Int64("activity.id", activity.Id),
							slog.String("activity.title", activity.Media.Title.UserPreferred),
							slog.String("stack", err.Error()),
						)
					} else {
						updateOccured = true
					}
					continue
				}

				// Next, we check if the existing chapter was the first in a streak ie; 102
				if err := database.Get(
					&existingItem,
					"SELECT * FROM db_media_items WHERE category = ? AND title = ? AND subtitle = ? ORDER BY created_at desc LIMIT 1",
					"manga",
					startChapter,
					activity.Media.Title.UserPreferred,
				); err == nil {
					err := updateChapter(database, activity, existingItem)
					if err != nil {
						slog.Error("Failed to update entry",
							slog.Int64("activity.id", activity.Id),
							slog.String("activity.title", activity.Media.Title.UserPreferred),
							slog.String("stack", err.Error()),
						)
					} else {
						updateOccured = true
					}
					continue
				}
			}

			image, extension, dominantColours, err := utils.ExtractImageContent(activity.Media.CoverImage.ExtraLarge)
			if err != nil {
				slog.Error("Failed to extract image content",
					slog.String("stack", err.Error()),
					slog.String("image_url", activity.Media.CoverImage.ExtraLarge),
				)
				return
			}

			discImage, guid := utils.BytesToGUIDLocation(image, extension)

			if err := saveCover(guid.String(), image, extension); err != nil {
				slog.Error("Failed to save cover for Anilist",
					slog.String("stack", err.Error()),
					slog.String("guid", guid.String()),
				)
				return
			}

			// We haven't seen this chapter update so we'll save it
			schema := `INSERT INTO db_media_items (created_at, title, subtitle, category, is_active, duration_ms, dominant_colours, source, image) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
			_, err = database.Exec(
				schema,
				activity.CreatedAt,
				activity.Progress,
				activity.Media.Title.UserPreferred,
				"manga",
				false,
				0,
				dominantColours,
				"anilist",
				discImage,
			)
			if err != nil {
				slog.Error("Failed to save DB entry",
					slog.String("stack", err.Error()),
					slog.String("title", activity.Media.Title.UserPreferred),
				)
				continue
			}
			updateOccured = true
		}
	}

	if updateOccured {
		var latestItem models.ComboDBMediaItem
		latestItem, err := store.GetByCategory("manga")
		if err == nil {
			// If we've read manga in the past but only just fetched updates, we don't consider this "live"
			// so only update the live player if nothing else is live and manga is more recent
			if !CurrentPlaybackItem.IsActive && latestItem.OccuredAt > CurrentPlaybackItem.CreatedAt {
				playingItem := models.MediaItem{
					CreatedAt:       latestItem.OccuredAt,
					Title:           latestItem.Title,
					Subtitle:        latestItem.Subtitle,
					Category:        latestItem.Category,
					Source:          latestItem.Source,
					Duration:        latestItem.Duration,
					DominantColours: latestItem.DominantColours,
					IsActive:        latestItem.IsActive,
					Image:           latestItem.Image,
				}
				byteStream := new(bytes.Buffer)
				json.NewEncoder(byteStream).Encode(playingItem)
				events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
				CurrentPlaybackItem = playingItem
			}
		}
	}
}

func updateChapter(database *sqlx.DB, activity models.Activity, existingItem models.ComboDBMediaItem) error {
	query := `UPDATE db_media_items SET created_at = ?, title = ? WHERE id = ?`
	_, err := database.Exec(
		query,
		activity.CreatedAt,
		activity.Progress,
		existingItem.ID,
	)
	return err
}
