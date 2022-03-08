package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/marcus-crane/gunslinger/jobs"
)

func renderJSONMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	res := map[string]string{"message": message}
	json.NewEncoder(w).Encode(res)
}

func Register(mux *http.ServeMux) *http.ServeMux {

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "Welcome to Gunslinger, my handy do-everything API.\nYou can find the source code on <a href=\"https://github.com/marcus-crane/gunslinger\">Github</a>\n")
	})

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the base of Gunslinger's various APIs")
	})

	mux.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the v1 endpoint of the API")
	})

	mux.HandleFunc("/api/v2", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the v2 endpoint of the API. There are no v2 endpoints at present")
	})

	mux.HandleFunc("/api/v3", func(w http.ResponseWriter, r *http.Request) {
		renderJSONMessage(w, "This is the v3 endpoint of the API")
	})

	mux.HandleFunc("/api/v3/playing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs.CurrentPlaybackItem)
	})

	// v1.Get("/videogames", handlers.GetGameInFocus)
	// v1.Post("/videogames", handlers.UpdateGameInFocus)
	// v1.Delete("/videogames", handlers.ClearGameInFocus)

	return mux
}
