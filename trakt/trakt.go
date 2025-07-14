package trakt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gregdel/pushover"
	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/playback"
	"github.com/marcus-crane/gunslinger/shared"
	"github.com/marcus-crane/gunslinger/utils"
)

const (
	accessTokenID           = "trakt:accesstoken"
	refreshTokenID          = "trakt:refreshtoken"
	traktOAuthAuthEndpoint  = "https://api.trakt.tv/oauth/authorize"
	traktOAuthTokenEndpoint = "https://api.trakt.tv/oauth/token"
	traktPlayingEndpoint    = "https://api.trakt.tv/users/sentry/watching"
	tmdbMovieEndpoint       = "https://api.themoviedb.org/3/movie/%d/images"
	tmdbEpisodeEndpoint     = "https://api.themoviedb.org/3/tv/%d/season/%d/episode/%d/images"
)

type NowPlayingResponse struct {
	ExpiresAt      string       `json:"expires_at"`
	StartedAt      string       `json:"started_at"`
	Action         string       `json:"action"`
	Type           string       `json:"type"`
	Movie          TraktSummary `json:"movie"`
	Episode        TraktEpisode `json:"episode"`
	Show           TraktSummary `json:"show"`
	PodcastEpisode TraktEpisode `json:"podcast_episode"`
	Podcast        TraktSummary `json:"podcast"`
}

type TraktEpisode struct {
	Season        int      `json:"season"`
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	IDs           TraktIDs `json:"ids"`
	Overview      string   `json:"overview"`
	OverviewPlain string   `json:"overview_plain"`
	Explicit      bool     `json:"explicit"`
	Runtime       int      `json:"runtime"`
}

type TraktSummary struct {
	Title         string   `json:"title"`
	Year          int      `json:"year"`
	IDs           TraktIDs `json:"ids"`
	Overview      string   `json:"overview"`
	OverviewPlain string   `json:"overview_plain"`
	Author        string   `json:"author"`
	Homepage      string   `json:"homepage"`
}

type TraktIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	TVDB  int    `json:"tvdb"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
	Apple int    `json:"apple"`
}

type TMDBImageResponse struct {
	ID      int         `json:"id"`
	Stills  []TMDBImage `json:"stills"`
	Posters []TMDBImage `json:"posters"`
}

type TMDBImage struct {
	FilePath string `json:"file_path"`
}

type TraktAccessTokenPayload struct {
	Code         string `json:"code"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectUri  string `json:"redirect_uri"`
	GrantType    string `json:"grant_type"`
}

type TraktRefreshTokenPayload struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectUri  string `json:"redirect_uri"`
	GrantType    string `json:"grant_type"`
}

func CheckForTokenRefresh(cfg config.Config, store db.Store) error {
	// TODO: Probably make this into some sort of scheduler for Trakt but
	// it'll do for now as we only need to refresh the token weekly
	tokenMetadata := store.GetTokenMetadataByID(accessTokenID)
	timeRemaining := time.Unix(tokenMetadata.CreatedAt, 0).Add(time.Second * time.Duration(tokenMetadata.ExpiresIn))
	// If the token is going to expire 1 day from now or less, we'll refresh early
	if time.Now().Add(time.Hour * 24).After(timeRemaining) {
		// Always assume token exists but we should not
		existingRefreshToken := store.GetTokenByID(refreshTokenID)
		token, err := refreshTokens(cfg, existingRefreshToken)
		if err != nil {
			slog.With("error", err).Error("failed to generate access tokens")
			return err
		}
		// Save our newly generated token
		if err := store.UpsertToken(accessTokenID, token.AccessToken); err != nil {
			slog.With("error", err).Error("failed to save access token")
			return err
		}
		if err := store.UpsertToken(refreshTokenID, token.RefreshToken); err != nil {
			slog.With("error", err).Error("failed to save refresh token")
			return err
		}
		if err := store.UpsertTokenMetadata(accessTokenID, token.CreatedAt, token.ExpiresIn); err != nil {
			slog.With("error", err).Error("failed to save access token metadata")
			return err
		}
	}
	return nil
}

func initialTokenFetch(cfg config.Config, store db.Store) (string, error) {
	token, err := performOAuth2Flow(cfg, 8082)
	if err != nil {
		slog.With("error", err).Error("failed to generate access tokens")
		return "", err
	}
	// Save our newly generated token
	if err := store.UpsertToken(accessTokenID, token.AccessToken); err != nil {
		slog.With("error", err).Error("failed to save access token")
		return "", err
	}
	if err := store.UpsertToken(refreshTokenID, token.RefreshToken); err != nil {
		slog.With("error", err).Error("failed to save refresh token")
		return "", err
	}
	if err := store.UpsertTokenMetadata(accessTokenID, token.CreatedAt, token.ExpiresIn); err != nil {
		slog.With("error", err).Error("failed to save access token metadata")
		return "", err
	}
	return token.AccessToken, nil
}

func performOAuth2Flow(cfg config.Config, port int) (*shared.TokenResponse, error) {
	state := shared.GenerateRandomString(16)
	ch := make(chan *shared.TokenResponse) // TODO: Make this shared
	var srv *http.Server

	pushoverApp := pushover.New(cfg.Pushover.Token)
	recipient := pushover.NewRecipient(cfg.Pushover.Recipient)

	// TODO: Ideally integrate with existing router to make life easier + support
	// possibly oauth refreshing server side (for bootstrapping)
	http.HandleFunc("/trakt/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		token, err := exchangeCodeForToken(cfg, code)
		if err != nil {
			http.Error(w, "Error exchanging code for token", http.StatusInternalServerError)
			return
		}
		ch <- token
		fmt.Fprintf(w, "Authentication successful! You can close this window.")
		go func() {
			time.Sleep(time.Second)
			srv.Shutdown(r.Context())
		}()
	})

	srv = &http.Server{Addr: fmt.Sprintf(":%d", port)}
	go func() { _ = srv.ListenAndServe() }()

	url := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&state=%s",
		traktOAuthAuthEndpoint, cfg.Trakt.ClientId, url.QueryEscape(cfg.Trakt.RedirectUri), state)

	slog.With(slog.String("url", url)).Info("Please open the following URL in your browser")
	message := &pushover.Message{
		Message:    "Refresh token has expired + you probably redeployed so we need to manually reauth",
		Title:      "Please auth with Trakt for Gunslinger",
		Priority:   pushover.PriorityHigh,
		URL:        url,
		URLTitle:   "Auth with Trakt",
		Timestamp:  time.Now().Unix(),
		DeviceName: "Gunslinger",
	}
	_, err := pushoverApp.SendMessage(message, recipient)
	if err != nil {
		fmt.Println(err)
		return &shared.TokenResponse{}, fmt.Errorf("failed to notify about oauth request")
	}

	token := <-ch
	return token, nil
}

func exchangeCodeForToken(cfg config.Config, code string) (*shared.TokenResponse, error) {
	payload := TraktAccessTokenPayload{
		Code:         code,
		ClientID:     cfg.Trakt.ClientId,
		ClientSecret: cfg.Trakt.ClientSecret,
		RedirectUri:  cfg.Trakt.RedirectUri,
		GrantType:    "authorization_code",
	}

	b, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", traktOAuthTokenEndpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var token shared.TokenResponse
	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func refreshTokens(cfg config.Config, refreshToken string) (shared.TokenResponse, error) {
	payload := TraktRefreshTokenPayload{
		RefreshToken: refreshToken,
		ClientID:     cfg.Trakt.ClientId,
		ClientSecret: cfg.Trakt.ClientSecret,
		RedirectUri:  cfg.Trakt.RedirectUri,
		GrantType:    "refresh_token",
	}

	b, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", traktOAuthAuthEndpoint, bytes.NewReader(b))
	if err != nil {
		return shared.TokenResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return shared.TokenResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return shared.TokenResponse{}, err
	}

	var newTokens shared.TokenResponse
	err = json.Unmarshal(body, &newTokens)
	if err != nil {
		return shared.TokenResponse{}, err
	}

	return newTokens, nil
}

func getArtFromTMDB(apiKey string, traktResponse NowPlayingResponse) (string, error) {
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
	var tmdbImageResponse TMDBImageResponse

	if err = json.Unmarshal(body, &tmdbImageResponse); err != nil {
		slog.Error("Failed to fetch image data from TMDB", slog.String("body", string(body)))
	}

	imagePath := ""
	if traktResponse.Type == "movie" {
		imagePath = tmdbImageResponse.Posters[0].FilePath
	} else {
		imagePath = tmdbImageResponse.Stills[0].FilePath
	}

	return fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", imagePath), nil
}

func GetCurrentlyPlaying(cfg config.Config, ps *playback.PlaybackSystem, client http.Client, store db.Store) {
	accessToken := store.GetTokenByID(accessTokenID)
	// We don't have an access token so let's request one
	if accessToken == "" {
		slog.Info("No trakt token found so prompting to OAuth")
		newAccessToken, err := initialTokenFetch(cfg, store)
		if err != nil {
			slog.Error("Failed to populate initial Trakt token", slog.String("error", err.Error()))
			return
		}
		accessToken = newAccessToken
	}
	req, err := http.NewRequest("HEAD", traktPlayingEndpoint, nil)
	if err != nil {
		slog.Error("Failed to build HEAD request for Trakt", slog.String("error", err.Error()))
		return
	}
	req.Header = http.Header{
		"Accept":            []string{"application/json"},
		"Authorization":     []string{fmt.Sprintf("Bearer %s", accessToken)},
		"Content-Type":      []string{"application/json"},
		"trakt-api-version": []string{"2"},
		"trakt-api-key":     []string{cfg.Trakt.ClientId},
		"User-Agent":        []string{utils.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to make HEAD request to Trakt",
			slog.String("error", err.Error()),
			slog.String("code", res.Status),
		)
		return
	}

	// Nothing is playing so we don't need to do any further work
	if res.StatusCode == 204 {
		ps.DeactivateBySource(string(playback.Trakt))
		return
	}

	// Do a proper, more expensive request now that we've got something fresh
	req2, err := http.NewRequest("GET", traktPlayingEndpoint, nil)
	if err != nil {
		slog.Error("Failed to build GET request for Trakt", slog.String("error", err.Error()))
		return
	}
	req2.Header = req.Header
	res2, err := client.Do(req2)
	if err != nil {
		slog.Error("Failed to make GET request for Trakt", slog.String("error", err.Error()))
		return
	}

	// Nothing is playing so we should check if anything needs to be cleaned up
	// or if we need to do a state transition
	if res2.StatusCode == 204 {
		ps.DeactivateBySource(string(playback.Trakt))
		return
	}

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		slog.Error("Failed to read Trakt response",
			slog.String("error", err.Error()),
			slog.String("code", res2.Status),
		)
		return
	}
	var traktResponse NowPlayingResponse

	if err = json.Unmarshal(body, &traktResponse); err != nil {
		// TODO: Check status code
		slog.Error("Failed to unmarshal Trakt response",
			slog.String("error", err.Error()),
			slog.String("code", res2.Status),
		)
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

	imageUrl, err := getArtFromTMDB(cfg.Trakt.TMDBToken, traktResponse)
	if err != nil {
		slog.Error("Failed to retrieve art from TMDB",
			slog.String("error", err.Error()),
		)
		return
	}
	image, extension, domColours, err := utils.ExtractImageContent(imageUrl)
	if err != nil {
		slog.Error("Failed to extract image content",
			slog.String("error", err.Error()),
			slog.String("image_url", imageUrl),
		)
		return
	}

	started, err := time.Parse("2006-01-02T15:04:05.999Z", traktResponse.StartedAt)
	if err != nil {
		slog.Error("Failed to parse Trakt start time",
			slog.String("error", err.Error()),
			slog.String("started_at", traktResponse.StartedAt),
		)
		return
	}
	ends, err := time.Parse("2006-01-02T15:04:05.999Z", traktResponse.ExpiresAt)
	if err != nil {
		slog.Error("Failed to parse Trakt expiry time",
			slog.String("error", err.Error()),
			slog.String("expires_at", traktResponse.ExpiresAt),
		)
		return
	}

	duration := int(ends.Sub(started).Milliseconds())

	update := playback.Update{
		MediaItem: playback.MediaItem{
			Category:        traktResponse.Type,
			Duration:        duration,
			Source:          string(playback.Trakt),
			DominantColours: domColours,
		},
		Elapsed: time.Since(started),
		Status:  playback.StatusPlaying,
	}

	if traktResponse.Type == "movie" {
		update.MediaItem.Title = traktResponse.Movie.Title
		update.MediaItem.Subtitle = fmt.Sprint(traktResponse.Movie.Year)
	} else {
		update.MediaItem.Title = fmt.Sprintf(
			"%02dx%02d %s",
			traktResponse.Episode.Season, // Season number
			traktResponse.Episode.Number, // Episode number
			traktResponse.Episode.Title,
		)
		update.MediaItem.Subtitle = traktResponse.Show.Title
	}

	hash := playback.GenerateMediaID(&update)
	coverUrl, err := utils.SaveCover(cfg, hash, image, extension)
	if err != nil {
		slog.Error("Failed to save cover for Trakt",
			slog.String("error", err.Error()),
			slog.String("guid", hash),
			slog.String("title", update.MediaItem.Title),
		)
	}

	update.MediaItem.Image = coverUrl

	if err := ps.UpdatePlaybackState(update); err != nil {
		slog.Error("Failed to save Trakt update",
			slog.String("error", err.Error()),
			slog.String("title", update.MediaItem.Title))
	}
}
