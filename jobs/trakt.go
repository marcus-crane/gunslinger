package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
	"github.com/r3labs/sse/v2"
)

var (
	traktPlayingEndpoint = "https://api.trakt.tv/users/sentry/watching"
	tmdbMovieEndpoint    = "https://api.themoviedb.org/3/movie/%d/images"
	tmdbEpisodeEndpoint  = "https://api.themoviedb.org/3/tv/%d/season/%d/episode/%d/images"
)

func getArtFromTMDB(apiKey string, traktResponse models.NowPlayingResponse) (string, error) {
	url := ""
	if traktResponse.Type == "movie" {
		url = fmt.Sprintf(tmdbMovieEndpoint, traktResponse.Movie.IDs.TMDB)
	} else {
		url = fmt.Sprintf(tmdbEpisodeEndpoint, traktResponse.Show.IDs.TMDB, traktResponse.Episode.Season, traktResponse.Episode.Number)
	}

	url = fmt.Sprintf("%s?api_key=%s", url, apiKey)

	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var tmdbImageResponse models.TMDBImageResponse

	if err = json.Unmarshal(body, &tmdbImageResponse); err != nil {
		fmt.Println("Error fetching Trakt data: ", err)
	}

	imagePath := ""
	if traktResponse.Type == "movie" {
		imagePath = tmdbImageResponse.Posters[0].FilePath
	} else {
		imagePath = tmdbImageResponse.Stills[0].FilePath
	}

	return fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", imagePath), nil
}

func GetCurrentlyPlayingTrakt(database *sqlx.DB) {
	traktBearerToken := utils.MustEnv("TRAKT_BEARER_TOKEN")
	traktClientID := utils.MustEnv("TRAKT_CLIENT_ID")
	tmdbToken := utils.MustEnv("TMDB_TOKEN")

	var client http.Client
	req, err := http.NewRequest("HEAD", traktPlayingEndpoint, nil)
	if err != nil {
		log.Printf("Failed to build HEAD request for Trakt: %+v\n", err)
		return
	}
	req.Header = http.Header{
		"Accept":            []string{"application/json"},
		"Authorization":     []string{fmt.Sprintf("Bearer %s", traktBearerToken)},
		"Content-Type":      []string{"application/json"},
		"trakt-api-version": []string{"2"},
		"trakt-api-key":     []string{traktClientID},
		"User-Agent":        []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to make HEAD request to Trakt: %+v\n", err)
		return
	}

	// Nothing is playing so we should check if anything needs to be cleaned up
	// or if we need to do a state transition
	if res.StatusCode == 204 {
		if CurrentPlaybackItem.IsActive && CurrentPlaybackItem.Source == "trakt" {
			CurrentPlaybackItem.IsActive = false
			byteStream := new(bytes.Buffer)
			json.NewEncoder(byteStream).Encode(CurrentPlaybackItem)
			events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		}
		return
	}

	// Do a proper, more expensive request now that we've got something fresh
	req2, err := http.NewRequest("GET", traktPlayingEndpoint, nil)
	if err != nil {
		log.Printf("Failed to build GET request for Trakt: %+v\n", err)
		return
	}
	req2.Header = req.Header
	res2, err := client.Do(req2)
	if err != nil {
		log.Printf("Failed to make GET request to Trakt: %+v\n", err)
		return
	}

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		log.Printf("Failed to read Trakt response: %+v\n", err)
		return
	}
	var traktResponse models.NowPlayingResponse

	if err = json.Unmarshal(body, &traktResponse); err != nil {
		// TODO: Check status code
		return
	}

	// We only want to use Trakt to capture movies and TV series that have been
	// manually checked into, to supplement that they aren't automatically being
	// captured by Plex polling. It's also wasteful to call out to TMDB and what
	// not if nothing is happening
	//
	// TODO: Handle checking for 204 and cleaning up any active resources
	if traktResponse.Action != "checkin" {
		return
	}

	imageUrl, err := getArtFromTMDB(tmdbToken, traktResponse)
	if err != nil {
		log.Printf("Failed to retrieve art from TMDB: %+v\n", err)
		return
	}
	image, extension, domColours, err := utils.ExtractImageContent(imageUrl)
	if err != nil {
		log.Printf("Failed to extract image content: %+v\n", err)
		return
	}
	imageLocation, guid := utils.BytesToGUIDLocation(image, extension)

	started, err := time.Parse("2006-01-02T15:04:05.999Z", traktResponse.StartedAt)
	if err != nil {
		log.Printf("Failed to parse Trakt start time: %s\n", traktResponse.StartedAt)
		return
	}
	ends, err := time.Parse("2006-01-02T15:04:05.999Z", traktResponse.ExpiresAt)
	if err != nil {
		log.Printf("Failed to parse Trakt expiry time: %s\n", traktResponse.ExpiresAt)
		return
	}

	duration := int(ends.Sub(started).Milliseconds())
	elapsed := int(time.Now().Sub(started).Milliseconds())

	playingItem := models.MediaItem{
		Category:        traktResponse.Type,
		IsActive:        true,
		Elapsed:         elapsed,
		Duration:        duration,
		Source:          "trakt",
		DominantColours: domColours,
		Image:           imageLocation,
	}

	// Fallback in case a scrobble somehow runs too long
	if time.Now().After(ends) {
		playingItem.IsActive = false
	}

	if traktResponse.Type == "movie" {
		playingItem.Title = traktResponse.Movie.Title
		playingItem.Subtitle = fmt.Sprint(traktResponse.Movie.Year)
	} else {
		playingItem.Title = fmt.Sprintf(
			"%02dx%02d %s",
			traktResponse.Episode.Season, // Season number
			traktResponse.Episode.Number, // Episode number
			traktResponse.Episode.Title,
		)
		playingItem.Subtitle = traktResponse.Show.Title
	}

	// reflect.DeepEqual is good enough for our purposes even though
	// it doesn't do things like properly copmare timestamp metadata.
	// For just checking if we should emit a message, it's good enough
	if !reflect.DeepEqual(CurrentPlaybackItem, playingItem) {
		byteStream := new(bytes.Buffer)
		json.NewEncoder(byteStream).Encode(playingItem)
		events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		// We want to make sure that we don't resave if the server restarts
		// to ensure the history endpoint is relatively accurate
		var previousItem models.ComboDBMediaItem
		if err := database.Get(
			&previousItem,
			"SELECT * FROM db_media_items WHERE category = ? ORDER BY created_at desc LIMIT 1",
			playingItem.Category,
		); err == nil || err.Error() == "sql: no rows in result set" {
			if CurrentPlaybackItem.Title != playingItem.Title && previousItem.Title != playingItem.Title {
				if err := saveCover(guid.String(), image, extension); err != nil {
					fmt.Printf("Failed to save cover for Trakt: %+v\n", err)
				}
				schema := `INSERT INTO db_media_items (created_at, title, subtitle, category, is_active, duration_ms, dominant_colours, source, image) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
				_, err := database.Exec(
					schema,
					time.Now().Unix(),
					playingItem.Title,
					playingItem.Subtitle,
					playingItem.Category,
					playingItem.IsActive,
					playingItem.Duration,
					playingItem.DominantColours,
					playingItem.Source,
					playingItem.Image,
				)
				if err != nil {
					fmt.Println("Failed to save DB entry")
					log.Print(err)
				}
			}
		} else {
			log.Print(err)
		}
	}

	CurrentPlaybackItem = playingItem
}
