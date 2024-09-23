package retroachievements

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	ProfileURL = "https://retroachievements.org/API/API_GetUserSummary.php?u=%s&g=1&a=2&y=%s"
)

type Profile struct {
	RichPresenceMsg string               `json:"RichPresenceMsg"`
	LastGameID      int                  `json:"LastGameID"`
	RecentlyPlayed  []RecentlyPlayedGame `json:"RecentlyPlayed"`
	LastGame        LastPlayedGame       `json:"LastGame"`
}

// TODO: Profile has embedded last game but doesn't seem populated?

type RecentlyPlayedList []RecentlyPlayedGame

type RecentlyPlayedGame struct {
	GameID     int    `json:"GameID"`
	LastPlayed string `json:"LastPlayed"`
}

type LastPlayedGame struct {
	ID          int    `json:"ID"`
	Title       string `json:"Title"`
	ImageBoxArt string `json:"ImageBoxArt"`
	Developer   string `json:"Developer"`
}

func GetCurrentlyPlaying(cfg config.Config, ps *playback.PlaybackSystem, client http.Client) {
	slog.Info("Processing Retroachievements")
	url := fmt.Sprintf(ProfileURL, cfg.RetroAchievements.Username, cfg.RetroAchievements.Token)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Failed to prepare RetroAchievements request",
			slog.String("stack", err.Error()),
		)
		return
	}
	req.Header = http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{utils.UserAgent},
	}
	slog.Info("RA: Built request. About to request")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Failed to contact RetroAchievements for updates",
			slog.String("stack", err.Error()),
		)
		return
	}
	slog.Info("Got RA response back", slog.String("status", res.Status))
	if res.StatusCode != 200 {
		slog.Error("Received a non-200 status code from Retroachievements",
			slog.String("status", res.Status),
		)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("Failed to read RetroAchievements response",
			slog.String("stack", err.Error()),
		)
		return
	}
	var raProfile Profile
	if err = json.Unmarshal(body, &raProfile); err != nil {
		slog.Error("Error fetching RetroAchievements data",
			slog.String("stack", err.Error()),
		)
		return
	}
	var lastPlayed RecentlyPlayedGame
	slog.Info("Found recently played titles",
		slog.Int("count", len(raProfile.RecentlyPlayed))
	)
	for i, game := range raProfile.RecentlyPlayed {
		if i == 0 {
			lastPlayed = game
		}
	}
	// Somehow we got nothing played recently
	if lastPlayed.GameID == 0 {
		slog.Info("Found no last played title for RA")
		ps.DeactivateBySource(string(playback.RetroAchievements))
		return
	}

	if raProfile.LastGame.ID != lastPlayed.GameID {
		slog.Info("Last played segment for RA didn't match latest entry in history list")
		// We know the last game but seemingly a newer game exists. We need the timestamp to know whether it's active.
		ps.DeactivateBySource(string(playback.RetroAchievements))
		return
	}

	lastSeen, err := time.Parse("2006-01-02 03:04:05", lastPlayed.LastPlayed)
	if err != nil {
		// We have no idea when this was last played so assume it was ages ago
		ps.DeactivateBySource(string(playback.RetroAchievements))
		return
	}

	slog.Info("Saw a recently played title on RA",
		slog.String("last_seen", lastPlayed.LastPlayed)
	)

	minutesSinceLastSeen := time.Now().UTC().Sub(lastSeen)

	if minutesSinceLastSeen.Minutes() >= 3 {
		slog.With(slog.String("last_seen", lastPlayed.LastPlayed), slog.String("minutes_passed", minutesSinceLastSeen.String())).Info("Not seen active on RA for period. Deactivating...")
		// If we haven't seen this game in at least 5 minutes, we assume we're not playing anymore.
		// RA appears to update each minute while connected via WiFi so this should be more than enough.
		ps.DeactivateBySource(string(playback.RetroAchievements))
		return
	}

	// 2024-09-23 10:12:39

	imageUrl := fmt.Sprintf("https://media.retroachievements.org%s", raProfile.LastGame.ImageBoxArt)
	slog.With(slog.String("image_url", imageUrl)).Info("Built image link")

	image, extension, domColours, err := utils.ExtractImageContent(imageUrl)
	if err != nil {
		slog.Error("Failed to extract image content",
			slog.String("stack", err.Error()),
			slog.String("image_url", imageUrl),
		)
		return
	}

	imageLocation, _ := utils.BytesToGUIDLocation(image, extension)

	update := playback.Update{
		MediaItem: playback.MediaItem{
			Title:           raProfile.LastGame.Title,
			Subtitle:        raProfile.LastGame.Developer,
			Category:        string(playback.Gaming),
			Duration:        0,
			Source:          string(playback.RetroAchievements),
			Image:           imageLocation,
			DominantColours: domColours,
		},
		Status: playback.StatusPlaying,
	}

	if err := ps.UpdatePlaybackState(update); err != nil {
		slog.Error("Failed to save RetroAchievements update",
			slog.String("stack", err.Error()),
			slog.String("title", update.MediaItem.Title))
		return
	}

	hash := playback.GenerateMediaID(&update)
	if err := utils.SaveCover(cfg, hash, image, extension); err != nil {
		slog.Error("Failed to save cover for RetroAchievements",
			slog.String("stack", err.Error()),
			slog.String("guid", hash),
			slog.String("title", update.MediaItem.Title),
		)
		return
	}
}
