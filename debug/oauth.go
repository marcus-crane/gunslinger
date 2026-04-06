package debug

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/marcus-crane/gunslinger/shared"
	"github.com/marcus-crane/gunslinger/spotify"
	"github.com/marcus-crane/gunslinger/trakt"
)

type pendingOAuth struct {
	provider  string
	token     string // debug auth token for redirect back
	createdAt time.Time
}

type oauthStateStore struct {
	mu     sync.Mutex
	states map[string]pendingOAuth
}

func newOAuthStateStore() *oauthStateStore {
	return &oauthStateStore{states: make(map[string]pendingOAuth)}
}

func (s *oauthStateStore) add(state, provider, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Clean up expired entries while we're here
	for k, v := range s.states {
		if time.Since(v.createdAt) > 10*time.Minute {
			delete(s.states, k)
		}
	}
	s.states[state] = pendingOAuth{
		provider:  provider,
		token:     token,
		createdAt: time.Now(),
	}
}

func (s *oauthStateStore) consume(state string) (pendingOAuth, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.states[state]
	if !ok {
		return pendingOAuth{}, false
	}
	delete(s.states, state)
	if time.Since(p.createdAt) > 10*time.Minute {
		return pendingOAuth{}, false
	}
	return p, true
}

func (h *Handler) ServeReauth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.FormValue("token")
	if token == "" || token != h.cfg.Gunslinger.SuperSecretToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	provider := r.FormValue("provider")

	var authURL string
	switch provider {
	case "spotify":
		authURL = fmt.Sprintf("%s?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&state=",
			"https://accounts.spotify.com/authorize",
			h.cfg.Spotify.ClientId,
			url.QueryEscape("streaming"),
			url.QueryEscape(h.cfg.Spotify.RedirectUri),
		)
	case "trakt":
		authURL = fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&state=",
			"https://api.trakt.tv/oauth/authorize",
			h.cfg.Trakt.ClientId,
			url.QueryEscape(h.cfg.Trakt.RedirectUri),
		)
	default:
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	state := shared.GenerateRandomString(16)
	h.oauthStates.add(state, provider, token)

	http.Redirect(w, r, authURL+state, http.StatusFound)
}

func (h *Handler) ServeOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	pending, ok := h.oauthStates.consume(state)
	if !ok {
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	var token *shared.TokenResponse
	var err error
	var accessTokenID, refreshTokenID string

	switch pending.provider {
	case "spotify":
		token, err = spotify.ExchangeCodeForToken(h.cfg, code)
		accessTokenID = "spotify:accesstoken"
		refreshTokenID = "spotify:refreshtoken"
	case "trakt":
		token, err = trakt.ExchangeCodeForToken(h.cfg, code)
		accessTokenID = "trakt:accesstoken"
		refreshTokenID = "trakt:refreshtoken"
	}

	if err != nil {
		slog.Error("Failed to exchange code for token",
			slog.String("provider", pending.provider),
			slog.String("error", err.Error()))
		redirectURL := fmt.Sprintf("/debug?token=%s&status=error&provider=%s",
			url.QueryEscape(pending.token), pending.provider)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	if err := h.store.UpsertToken(accessTokenID, token.AccessToken); err != nil {
		slog.Error("Failed to save access token", slog.String("error", err.Error()))
	}
	if err := h.store.UpsertToken(refreshTokenID, token.RefreshToken); err != nil {
		slog.Error("Failed to save refresh token", slog.String("error", err.Error()))
	}

	createdAt := token.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	if err := h.store.UpsertTokenMetadata(accessTokenID, createdAt, token.ExpiresIn); err != nil {
		slog.Error("Failed to save token metadata", slog.String("error", err.Error()))
	}

	slog.Info("Successfully reauthorized via debug page",
		slog.String("provider", pending.provider))

	redirectURL := fmt.Sprintf("/debug?token=%s&status=ok&provider=%s",
		url.QueryEscape(pending.token), pending.provider)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

