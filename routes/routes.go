package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/cors"
	"gorm.io/gorm"

	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/models"
)

func renderJSONMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	res := map[string]string{"message": message}
	json.NewEncoder(w).Encode(res)
}

func Register(mux *http.ServeMux, database *gorm.DB) http.Handler {

	events.Server.CreateStream("playback")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Welcome to Gunslinger, my handy do-everything API.\nYou can find the source code on <a href=\"https://github.com/marcus-crane/gunslinger\">Github</a>\n")
	})

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the base of Gunslinger's various APIs")
	})

	mux.HandleFunc("/api/v3", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the v3 endpoint of the API")
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
		var result []models.DBMediaItem
		// If nothing is playing, the "now playing" will likely be the same as the
		// first history item so we skip it if now playing and index 0 of history match.
		// We don't fully do an offset jump though as an item is only committed to the DB
		// when it changes to inactive so we don't want to hide a valid item in that state
		database.Limit(7).Order("created_at desc").Find(&result)
		var response []models.ResponseMediaItem
		for idx, item := range result {
			// A valid case is when I just listen to the same song over and over so
			// we need to ensure we're in the right state to skip historical items
			if idx == 0 && item.Title == jobs.CurrentPlaybackItem.Title && jobs.CurrentPlaybackItem.IsActive {
				continue
			}
			rItem := models.ResponseMediaItem{
				OccuredAt: item.CreatedAt.Format(time.RFC3339),
				Title:     item.Title,
				Subtitle:  item.Subtitle,
				Category:  item.Category,
				Source:    item.Source,
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
