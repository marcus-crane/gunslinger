package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	hmacext "github.com/alexellis/hmac/v2"
	"github.com/antchfx/htmlquery"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/rs/cors"

	"github.com/marcus-crane/gunslinger/beeminder"
	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/readwise"
	"github.com/marcus-crane/gunslinger/utils"
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

var requestsRe = regexp.MustCompile(`(?m).*Marcus\W+(\d+).*requests`)

func renderJSONMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	res := map[string]string{"message": message}
	json.NewEncoder(w).Encode(res)
}

func RegisterRoutes(mux *http.ServeMux, cfg config.Config, ps *playback.PlaybackSystem) http.Handler {

	events.Server.CreateStream("playback")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Welcome to Gunslinger, my handy do-everything API.\nYou can find the source code on <a href=\"https://github.com/marcus-crane/gunslinger\">Github</a>\n")
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Properly set up text/template and what not
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "User-agent: *\nDisallow: /")
	})

	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		cover := strings.ReplaceAll(r.URL.Path, "/static/", "")
		// plex:track:8080643347135712210.jpeg
		// translated into plex.track.<id>.jpeg internally as colons are valid in URIs but not all filesystems
		coverSegments := strings.Split(cover, ".")
		if len(coverSegments) != 4 {
			slog.With(slog.String("cover", cover)).Error("Request for cover art with invalid segment length")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		filename := fmt.Sprintf("%s.%s.%s", coverSegments[0], coverSegments[1], coverSegments[2])
		extension := coverSegments[3]
		image, err := utils.LoadCover(cfg, filename, extension)
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

	mux.HandleFunc("/api/v4", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the v4 endpoint of the API")
	})

	mux.HandleFunc("/api/v4/miniflux", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		rwToken := cfg.Readwise.Token
		if rwToken == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "this endpoint is not properly configured"})
			return
		}

		minifluxSecret := cfg.Miniflux.WebhookSecret
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

		// Miniflux has native support for Readwise Reader but I'd like to
		// do some extra stuff like rendering out sites that use JS to fetch
		// content which Miniflux and Readwise Reader do not play well with at times

		var largestImageUrl string
		var largestImageSize int64

		for _, e := range minifluxPayload.Entry.Enclosures {
			if strings.HasPrefix(e.MimeType, "image/") {
				if e.Size > largestImageSize {
					largestImageUrl = e.URL
				}
			}
		}

		var content string

		// nzh site sucks so using a very cool proxy that requires js rendering :^)
		if strings.Contains(minifluxPayload.Entry.URL, "nzherald.co.nz/") {
			ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf), chromedp.WithErrorf(log.Printf))
			defer cancel()

			ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			slog.Info("spinning up chrome headless")

			url := fmt.Sprintf("https://nzhp.and.nz/%s", minifluxPayload.Entry.URL)
			slog.With("url", url).Info("scraping site...")
			err := chromedp.Run(ctx, chromedp.Navigate(url), chromedp.WaitVisible(`h1.article__heading`), chromedp.Evaluate("let node = document.querySelector('#header'); node.parentNode.removeChild(node)", nil), chromedp.ActionFunc(func(ctx context.Context) error {
				node, err := dom.GetDocument().Do(ctx)
				if err != nil {
					return err
				}
				content, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
				return err
			}))
			if err != nil {
				slog.With(slog.Any("error", err)).With(slog.String("url", url)).Error("failed to scrape site")
				json.NewEncoder(w).Encode(map[string]string{"error": "failed to parse request"})
				return
			}

			slog.Info("successfully scraped site")
		}

		if content == "" {
			content = minifluxPayload.Entry.Content
		}

		payload := readerPayload{
			HTML:            content,
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

		rwToken := cfg.Readwise.Token
		if rwToken == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "this endpoint is not properly configured"})
			return
		}

		ingestSecret := cfg.Gunslinger.SuperSecretToken
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
		if len(ps.State) == 0 {
			// If nothing is playing, we'll return the most recent item
			// TODO: Should return all that were playing? Maybe not
			results, err := ps.GetHistory(1)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			if len(results) == 0 {
				json.NewEncoder(w).Encode([]string{})
				return
			}
			for i, result := range results {
				results[i].Image = "/static/" + strings.ReplaceAll(result.ID, ":", ".") + ".jpeg"
			}
			json.NewEncoder(w).Encode(results)
			return
		}
		mutatingState := ps.State
		for i, result := range mutatingState {
			mutatingState[i].Image = "/static/" + strings.ReplaceAll(result.ID, ":", ".") + ".jpeg"
		}
		json.NewEncoder(w).Encode(mutatingState)
	})

	mux.HandleFunc("/api/v4/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		results, err := ps.GetHistory(7)
		// If nothing is playing, the "now playing" will likely be the same as the
		// first history item so we skip it if now playing and index 0 of history match.
		// We don't fully do an offset jump though as an item is only committed to the DB
		// when it changes to inactive so we don't want to hide a valid item in that state
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if len(results) == 0 {
			json.NewEncoder(w).Encode([]string{})
			return
		}
		for i, result := range results {
			results[i].Image = "/static/" + strings.ReplaceAll(result.ID, ":", ".") + ".jpeg"
		}
		json.NewEncoder(w).Encode(results)
	})

	mux.HandleFunc("/api/v4/readwise/tags", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Gunslinger.SuperSecretToken == "" {
			renderJSONMessage(w, "This endpoint is misconfigured and can not be used currently")
			return
		}
		qVal := r.URL.Query()
		if !qVal.Has("token") {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if qVal.Get("token") != cfg.Gunslinger.SuperSecretToken {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if r.Method != http.MethodGet {
			renderJSONMessage(w, "That method is invalid for this endpoint")
			return
		}
		tags, err := readwise.CountTags(cfg)
		if err != nil {
			renderJSONMessage(w, "Something went wrong trying to count tags")
			return
		}
		json.NewEncoder(w).Encode(tags)
	})

	mux.HandleFunc("/api/v4/readwise/document_counts", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Gunslinger.SuperSecretToken == "" {
			renderJSONMessage(w, "This endpoint is misconfigured and can not be used currently")
			return
		}
		qVal := r.URL.Query()
		if !qVal.Has("token") {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if qVal.Get("token") != cfg.Gunslinger.SuperSecretToken {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if r.Method != http.MethodGet {
			renderJSONMessage(w, "That method is invalid for this endpoint")
			return
		}
		documentCounts, err := readwise.GetDocumentCounts(cfg)
		if err != nil {
			renderJSONMessage(w, "Something went wrong trying to count documents")
			return
		}
		json.NewEncoder(w).Encode(documentCounts)
	})

	mux.HandleFunc("/api/v4/item", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Gunslinger.SuperSecretToken == "" {
			renderJSONMessage(w, "This endpoint is misconfigured and can not be used currently")
			return
		}
		qVal := r.URL.Query()
		if !qVal.Has("token") {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if qVal.Get("token") != cfg.Gunslinger.SuperSecretToken {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if r.Method != http.MethodDelete {
			renderJSONMessage(w, "That method is invalid for this endpoint")
			return
		}
		if !qVal.Has("playback_id") {
			renderJSONMessage(w, "An ID did not appear to be provided")
			return
		}
		playback_id, err := strconv.ParseInt(qVal.Get("playback_id"), 10, 0)
		if err != nil {
			renderJSONMessage(w, "That ID could not be converted into an integer")
			return
		}
		if err := ps.DeleteItem(int(playback_id)); err != nil {
			renderJSONMessage(w, "Something went wrong trying to delete that item")
			return
		}
		renderJSONMessage(w, "Operation was successfully executed")
	})

	mux.HandleFunc("/beeminder/oias", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Gunslinger.SuperSecretToken == "" {
			renderJSONMessage(w, "This endpoint is misconfigured and can not be used currently")
			return
		}
		qVal := r.URL.Query()
		if !qVal.Has("token") {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if qVal.Get("token") != cfg.Gunslinger.SuperSecretToken {
			renderJSONMessage(w, "Your request was not authorized")
			return
		}
		if r.Method != http.MethodPost {
			renderJSONMessage(w, "That method is invalid for this endpoint")
			return
		}
		doc, err := htmlquery.LoadURL("https://fyi.org.nz/categorise/play")
		if err != nil {
			panic(err)
		}
		topLeaders := htmlquery.FindOne(doc, "//body/div[1]/div[4]/div/div[1]/div[1]/table[2]")
		match := requestsRe.FindAllStringSubmatch(htmlquery.InnerText(topLeaders), -1)
		matchCount := match[0][1]
		err = beeminder.SubmitDatapoint(cfg, "oiacategorisation", matchCount, fmt.Sprintf("%s requests categorised", matchCount))
		if err != nil {
			w.WriteHeader(422)
		}
		w.WriteHeader(200)
	})

	mux.HandleFunc("/events", events.Server.ServeHTTP)

	mux.HandleFunc("/glance", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the glance endpoint of the API")
	})

	mux.HandleFunc("/glance/beeminder", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Widget-Title", "Beeminder")
		w.Header().Set("Widget-Content-Type", "html")
		w.Header().Set("Content-Type", "text/html")
		if cfg.Gunslinger.SuperSecretToken == "" {
			w.Write([]byte("This endpoint is misconfigured and can not be used currently"))
			return
		}
		qVal := r.URL.Query()
		if !qVal.Has("token") {
			w.Write([]byte("Your request was not authorized"))
			return
		}
		if qVal.Get("token") != cfg.Gunslinger.SuperSecretToken {
			w.Write([]byte("Your request was not authorized"))
			return
		}
		if r.Method != http.MethodGet {
			w.Write([]byte("That method is invalid for this endpoint"))
			return
		}
		goals, err := beeminder.FetchGoals(cfg)
		if err != nil {
			w.Write([]byte("Something went wrong trying to fetch Beeminder goals"))
			return
		}
		resp := `<ul class="list collapsible-container" data-collapse-after="5">`
		for _, goal := range goals {
			colorClass := "" // defaults to a normal colour
			if goal.RoadStatusColor == "red" {
				colorClass = "color-negative"
			}
			if goal.RoadStatusColor == "orange" {
				colorClass = "color-primary"
			}
			if goal.RoadStatusColor == "blue" {
				colorClass = "color-highlight"
			}
			if goal.RoadStatusColor == "green" {
				colorClass = "color-subdue"
			}
			goalUrl := fmt.Sprintf("https://www.beeminder.com/utf9k/%s", goal.Slug)
			resp += `<li><div class="flex gap-10 row-reverse-on-mobile thumbnail-parent">`
			resp += fmt.Sprintf(`<a href="%s"><img class="forum-post-list-thumbnail thumbnail loaded finished-transition" src="%s" loading="lazy"></a>`, goal.GraphUrl, goal.ThumbUrl)
			resp += fmt.Sprintf(`<div class="grow min-width-0"><a class="size-title-dynamic %s" href="%s" target="_blank" rel="noreferrer">%s</a>`, colorClass, goalUrl, goal.Slug)
			resp += fmt.Sprintf(`<ul class="list-horizontal-text"><li>%s</li><li>$%d</li></ul>`, goal.SafeSum, int(goal.Contract.Amount))
			resp += `</div></div></li>`
		}
		resp += `</ul>`
		w.Write([]byte(resp))
	})

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://utf9k.net", "http://localhost:1313", "https://b.utf9k.net", "https://next.utf9k.net"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Origin, Content-Type, Accept"},
	})

	handler := c.Handler(mux)

	return handler
}
