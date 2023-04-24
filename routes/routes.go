package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/cors"

	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/models"
)

func renderJSONMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	res := map[string]string{"message": message}
	json.NewEncoder(w).Encode(res)
}

func Register(mux *http.ServeMux, database *sqlx.DB) http.Handler {

	events.Server.CreateStream("playback")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Welcome to Gunslinger, my handy do-everything API.\nYou can find the source code on <a href=\"https://github.com/marcus-crane/gunslinger\">Github</a>\n")
	})

	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		cover := strings.Trim(r.URL.Path, "/static/")
		coverSegments := strings.Split(cover, ".")
		if len(coverSegments) != 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		guid := coverSegments[1]
		extension := coverSegments[2]
		image, err := jobs.LoadCover(guid, extension)
		if err != nil {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31622400")
		w.Header().Set("Content-Type", fmt.Sprintf("image/%s", extension))
		w.Write([]byte(image))
	})

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the base of Gunslinger's various APIs")
	})

	mux.HandleFunc("/api/v3", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the v3 endpoint of the API")
	})

	mux.HandleFunc("/api/v4", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the v4 endpoint of the API")
	})

	mux.HandleFunc("/api/v3/playing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs.CurrentPlaybackItem)
	})

	mux.HandleFunc("/api/v3/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&events.Sessions{SessionsSeen: events.SessionsSeen, ActiveSessions: events.ActiveSessions})
	})

	mux.HandleFunc("/api/v3/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var response []models.ResponseMediaItem
		var result []models.DBMediaItem
		// If nothing is playing, the "now playing" will likely be the same as the
		// first history item so we skip it if now playing and index 0 of history match.
		// We don't fully do an offset jump though as an item is only committed to the DB
		// when it changes to inactive so we don't want to hide a valid item in that state
		if err := database.Select(&result, "SELECT * FROM db_media_items ORDER BY created_at desc LIMIT 7"); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		for idx, item := range result {
			// A valid case is when I just listen to the same song over and over so
			// we need to ensure we're in the right state to skip historical items
			if idx == 0 && item.Title == jobs.CurrentPlaybackItem.Title && jobs.CurrentPlaybackItem.Backfilled {
				continue
			}
			rItem := models.ResponseMediaItem{
				OccuredAt: time.Unix(item.CreatedAt, 0).Format(time.RFC3339),
				Title:     item.Title,
				Subtitle:  item.Subtitle,
				Category:  item.Category,
				Source:    item.Source,
				Image:     item.Image,
			}
			response = append(response, rItem)
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/v4/playing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var result models.DBMediaItem
		if err := database.Get(&result, "SELECT * FROM db_media_items ORDER BY created_at desc LIMIT 1"); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		playbackItems := []models.ResponseMediaItem{
			{
				OccuredAt:       time.Unix(result.CreatedAt, 0).Format(time.RFC3339),
				Title:           result.Title,
				Subtitle:        result.Subtitle,
				Category:        result.Category,
				Source:          result.Source,
				Image:           result.Image,
				Duration:        result.DurationMs,
				DominantColours: result.DominantColours,
			},
		}
		json.NewEncoder(w).Encode(playbackItems)
	})

	mux.HandleFunc("/api/v4/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var response []models.ResponseMediaItem
		var result []models.DBMediaItem
		// If nothing is playing, the "now playing" will likely be the same as the
		// first history item so we skip it if now playing and index 0 of history match.
		// We don't fully do an offset jump though as an item is only committed to the DB
		// when it changes to inactive so we don't want to hide a valid item in that state
		if err := database.Select(&result, "SELECT * FROM db_media_items ORDER BY created_at desc LIMIT 7"); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		for idx, item := range result {
			// A valid case is when I just listen to the same song over and over so
			// we need to ensure we're in the right state to skip historical items
			if idx == 0 && item.Title == jobs.CurrentPlaybackItem.Title && jobs.CurrentPlaybackItem.Backfilled {
				continue
			}
			rItem := models.ResponseMediaItem{
				OccuredAt:       time.Unix(item.CreatedAt, 0).Format(time.RFC3339),
				Title:           item.Title,
				Subtitle:        item.Subtitle,
				Category:        item.Category,
				Source:          item.Source,
				Image:           item.Image,
				Duration:        item.DurationMs,
				DominantColours: item.DominantColours,
			}
			response = append(response, rItem)
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/events", events.Server.ServeHTTP)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://utf9k.net", "http://localhost:1313", "http://localhost:8080", "https://utf9k.pages.dev"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Origin, Content-Type, Accept"},
	})

	handler := c.Handler(mux)

	return handler
}
