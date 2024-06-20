package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	hmacext "github.com/alexellis/hmac/v2"
	"github.com/rs/cors"

	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/jobs"
	"github.com/marcus-crane/gunslinger/models"
)

type readerPayload struct {
	URL             string   `json:"url,omitempty"`
	HTML            string   `json:"html,omitempty"`
	ShouldCleanHTML bool     `json:"should_clean_html,omitempty"`
	Title           string   `json:"title,omitempty"`
	Author          string   `json:"author,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	PublishedDate   string   `json:"published_date,omitempty"`
	ImageURL        string   `json:"image_url,omitempty"`
	Location        string   `json:"location,omitempty"`
	Category        string   `json:"category,omitempty"`
	SavedUsing      string   `json:"saved_using,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Notes           string   `json:"notes,omitempty"`
}

func renderJSONMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	res := map[string]string{"message": message}
	json.NewEncoder(w).Encode(res)
}

func Register(mux *http.ServeMux, store db.Store) http.Handler {

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
		// If nothing is playing, the "now playing" will likely be the same as the
		// first history item so we skip it if now playing and index 0 of history match.
		// We don't fully do an offset jump though as an item is only committed to the DB
		// when it changes to inactive so we don't want to hide a valid item in that state
		result, err := store.GetRecent()
		if err != nil {
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
				OccuredAt:       time.Unix(item.OccuredAt, 0).Format(time.RFC3339),
				Title:           item.Title,
				Subtitle:        item.Subtitle,
				Category:        item.Category,
				Source:          item.Source,
				Image:           item.Image,
				Duration:        item.Duration,
				DominantColours: item.DominantColours,
			}
			response = append(response, rItem)
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/v4/miniflux", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		rwToken := os.Getenv("READWISE_TOKEN")
		if rwToken == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "this endpoint is not properly configured"})
			return
		}

		minifluxSecret := os.Getenv("MINIFLUX_WEBHOOK_SECRET")
		if minifluxSecret == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "this endpoint is not properly configured"})
			return
		}

		if r.Header.Get("X-Miniflux-Event-Type") != "save_entry" {
			json.NewEncoder(w).Encode(map[string]string{"error": "this event type is not supported"})
			return
		}

		signature := r.Header.Get("X-Miniflux-Signature")

		slog.With(slog.String("signature", signature)).Info("Received signature")

		if signature == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "no signature was provided"})
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to read request body as part of signature validation"})
			return
		}

		if err := hmacext.Validate(body, fmt.Sprintf("sha256=%s", signature), minifluxSecret); err != nil {
			slog.With(slog.Any("error", err)).Error("Failed signature validation")
			json.NewEncoder(w).Encode(map[string]string{"error": "signature failed validation"})
			return
		}

		var minifluxPayload models.MinifluxSavedEntry

		if err := json.Unmarshal(body, &minifluxPayload); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to unmarshal request body"})
			return
		}

		var largestImageUrl string
		var largestImageSize int64

		for _, e := range minifluxPayload.Entry.Enclosures {
			if strings.HasPrefix(e.MimeType, "image/") {
				if e.Size > largestImageSize {
					largestImageUrl = e.URL
				}
			}
		}

		payload := readerPayload{
			HTML:            minifluxPayload.Entry.Content,
			URL:             minifluxPayload.Entry.URL,
			ShouldCleanHTML: true,
			ImageURL:        largestImageUrl,
			SavedUsing:      "Gunslinger",
		}

		data, err := json.Marshal(payload)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to marshal payload"})
			return
		}

		req, err := http.NewRequest("POST", "https://readwise.io/api/v3/save/", bytes.NewReader(data))
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to build request"})
			return
		}

		req.Header.Add("Authorization", fmt.Sprintf("Token %s", rwToken))
		req.Header.Add("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to send request"})
			return
		}
		defer resp.Body.Close()

		slog.With(slog.String("status", resp.Status), slog.String("url", payload.URL)).Info("Sent URL to Readwise")

		json.NewEncoder(w).Encode(map[string]string{"status": resp.Status})
	})

	// Yes, this is garbage and doesn't deserve to be put in here
	// It works for now though
	mux.HandleFunc("/api/v4/readwise_ingest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		rwToken := os.Getenv("READWISE_TOKEN")
		if rwToken == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "this endpoint is not properly configured"})
			return
		}

		ingestSecret := os.Getenv("INGEST_SECRET")
		if ingestSecret == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "this endpoint is not properly configured"})
			return
		}

		qVal := r.URL.Query()

		if qVal.Get("token") != ingestSecret {
			json.NewEncoder(w).Encode(map[string]string{"error": "ingest secret was incorrect"})
			return
		}

		url := qVal.Get("url")

		if url == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "url was not provided"})
			return
		}

		payload := readerPayload{
			URL:        url,
			SavedUsing: "Gunslinger",
		}

		data, err := json.Marshal(payload)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to marshal payload"})
			return
		}

		req, err := http.NewRequest("POST", "https://readwise.io/api/v3/save/", bytes.NewReader(data))
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to build request"})
			return
		}

		req.Header.Add("Authorization", fmt.Sprintf("Token %s", rwToken))
		req.Header.Add("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to send request"})
			return
		}
		defer resp.Body.Close()

		json.NewEncoder(w).Encode(map[string]string{"status": resp.Status})
	})

	mux.HandleFunc("/api/v4/playing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		result, err := store.GetNewest()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		playbackItems := []models.ComboDBMediaItem{
			{
				OccuredAt:       result.OccuredAt,
				Title:           result.Title,
				Subtitle:        result.Subtitle,
				Category:        result.Category,
				IsActive:        jobs.CurrentPlaybackItem.IsActive,
				Source:          result.Source,
				Image:           result.Image,
				Elapsed:         jobs.CurrentPlaybackItem.Elapsed,
				Duration:        result.Duration,
				DominantColours: result.DominantColours,
			},
		}
		json.NewEncoder(w).Encode(playbackItems)
	})

	mux.HandleFunc("/api/v4/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var response []models.ComboDBMediaItem
		result, err := store.GetRecent()
		// If nothing is playing, the "now playing" will likely be the same as the
		// first history item so we skip it if now playing and index 0 of history match.
		// We don't fully do an offset jump though as an item is only committed to the DB
		// when it changes to inactive so we don't want to hide a valid item in that state
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		for idx, item := range result {
			// A valid case is when I just listen to the same song over and over so
			// we need to ensure we're in the right state to skip historical items
			if idx == 0 && item.Title == jobs.CurrentPlaybackItem.Title && jobs.CurrentPlaybackItem.Backfilled {
				continue
			}
			rItem := models.ComboDBMediaItem{
				ID:              item.ID,
				OccuredAt:       item.OccuredAt,
				Title:           item.Title,
				Subtitle:        item.Subtitle,
				Category:        item.Category,
				Source:          item.Source,
				Image:           item.Image,
				Duration:        item.Duration,
				DominantColours: item.DominantColours,
			}
			response = append(response, rItem)
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/api/v4/item", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("SUPER_SECRET_TOKEN") == "" {
			renderJSONMessage(w, "This endpoint is misconfigured and can not be used currently")
			return
		}
		qVal := r.URL.Query()
		if !qVal.Has("token") {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if qVal.Get("token") != os.Getenv("SUPER_SECRET_TOKEN") {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if r.Method != http.MethodDelete {
			renderJSONMessage(w, "That method is invalid for this endpoint")
			return
		}
		if !qVal.Has("id") {
			renderJSONMessage(w, "An ID did not appear to be provided")
			return
		}
		id := qVal.Get("id")
		err := store.ExecCustom("DELETE FROM db_media_items WHERE id = ?", id)
		if err != nil {
			renderJSONMessage(w, "Something went wrong trying to delete that item")
			return
		}
		renderJSONMessage(w, "Operation was successfully executed")
	})

	mux.HandleFunc("/events", events.Server.ServeHTTP)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://utf9k.net", "http://localhost:1313", "http://localhost:8080", "https://utf9k.pages.dev", "https://b.utf9k.net"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Origin, Content-Type, Accept"},
	})

	handler := c.Handler(mux)

	return handler
}
