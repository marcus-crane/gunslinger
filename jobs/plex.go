package jobs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"reflect"

	color_extractor "github.com/marekm4/color-extractor"
	"github.com/r3labs/sse/v2"
	"gorm.io/gorm"

	"github.com/marcus-crane/gunslinger/events"
	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	plexSessionEndpoint = "/status/sessions"
	UserAgent           = "Gunslinger/1.0 (gunslinger@utf9k.net)"
)

func buildPlexURL(endpoint string) string {
	plexHostURL := utils.MustEnv("PLEX_URL")
	plexToken := utils.MustEnv("PLEX_TOKEN")
	return fmt.Sprintf("%s%s?X-Plex-Token=%s", plexHostURL, endpoint, plexToken)
}

func extractImageContent(imageUrl string) (string, []string) {
	var client http.Client
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		panic(err)
	}
	req.Header = http.Header{
		"User-Agent": []string{UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	var buf bytes.Buffer
	tee := io.TeeReader(res.Body, &buf)

	body, err := io.ReadAll(tee)
	if err != nil {
		panic(err)
	}

	var base64Encoding string

	mimeType := http.DetectContentType(body)

	switch mimeType {
	case "image/jpeg":
		base64Encoding += "data:image/jpeg;base64,"
	case "image/png":
		base64Encoding += "data:image/png;base64,"
	}

	base64Encoding += base64.StdEncoding.EncodeToString(body)

	var domColours []string

	image, _, _ := image.Decode(&buf)
	colours := color_extractor.ExtractColors(image)
	for _, c := range colours {
		domColours = append(domColours, ColorToHexString(c))
	}

	return base64Encoding, domColours
}

func ColorToHexString(c color.Color) string {
	r, g, b, a := c.RGBA()
	rgba := color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
	return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)

}

func GetCurrentlyPlayingPlex(database *gorm.DB) {
	sessionURL := buildPlexURL(plexSessionEndpoint)
	var client http.Client
	req, err := http.NewRequest("GET", sessionURL, nil)
	if err != nil {
		panic(err)
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{UserAgent},
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
	var plexResponse models.PlexResponse

	if err = json.Unmarshal(body, &plexResponse); err != nil {
		fmt.Println("Error fetching Plex data: ", err)
	}

	index := 0

	containsPlayingItem := false
	if plexResponse.MediaContainer.Size > 0 {
		for idx, entry := range plexResponse.MediaContainer.Metadata {
			if entry.Player.State == "playing" {
				containsPlayingItem = true
				// We may have multiple items in our queue at once
				// For example, a paused song while watching a TV show
				// so we need to figure out which item (if any) is the one
				// to surface
				index = idx
			}
		}
	}
	if !containsPlayingItem {
		// We may have removed the item entirely from the play queue so it won't
		// be in the API but we know if the source is Plex and nothing in Plex
		// is playing (it would be in the API if it were) then it's safe to
		// mark it as inactive
		if CurrentPlaybackItem.IsActive && CurrentPlaybackItem.Source == "plex" {
			CurrentPlaybackItem.IsActive = false
			// reflect.DeepEqual is good enough for our purposes even though
			// it doesn't do things like properly copmare timestamp metadata.
			// For just checking if we should emit a message, it's good enough
			byteStream := new(bytes.Buffer)
			json.NewEncoder(byteStream).Encode(CurrentPlaybackItem)
			events.Server.Publish("playback", &sse.Event{Data: byteStream.Bytes()})
		}
		return
	}

	mediaItem := plexResponse.MediaContainer.Metadata[index]
	thumbnail := mediaItem.Thumb

	// Tracks generally don't have a unique cover so we should use the album cover instead
	// This should hold true even for singles though
	if mediaItem.Type == "track" {
		thumbnail = mediaItem.ParentThumb
	}

	thumbnailUrl := buildPlexURL(thumbnail)
	imageB64, domColours := extractImageContent(thumbnailUrl)

	playingItem := models.MediaItem{
		Title:    mediaItem.Title,
		Category: mediaItem.Type,
		Elapsed:  mediaItem.ViewOffset,
		Duration: mediaItem.Duration,
		Source:   "plex",
		// TODO: Make use of the transcode endpoint or pull the thumbnail onto disc for caching
		Image:           imageB64,
		DominantColours: domColours,
	}

	if mediaItem.Player.State == "playing" {
		playingItem.IsActive = true
	}

	if mediaItem.Type == "episode" {
		playingItem.Title = fmt.Sprintf(
			"%02dx%02d %s",
			mediaItem.ParentIndex, // Season number
			mediaItem.Index,       // Episode number
			mediaItem.Title,
		)
	}

	if mediaItem.Type == "movie" {
		playingItem.Subtitle = mediaItem.Director[0].Name
	} else {
		playingItem.Subtitle = mediaItem.GrandparentTitle
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
		var previousItem models.DBMediaItem
		database.Where("category = ?", playingItem.Category).Last(&previousItem)
		if CurrentPlaybackItem.Title != playingItem.Title && previousItem.Title != playingItem.Title {
			dbItem := models.DBMediaItem{
				Title:    playingItem.Title,
				Subtitle: playingItem.Subtitle,
				Category: playingItem.Category,
				IsActive: playingItem.IsActive,
				Source:   playingItem.Source,
			}
			database.Save(&dbItem)
			if err := saveCover(playingItem.Image, playingItem.Category); err != nil {
				fmt.Println("Failed to save cover for Plex")
			}
		}
	}

	CurrentPlaybackItem = playingItem
}
