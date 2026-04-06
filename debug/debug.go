package debug

import (
	"html/template"
	"net/http"
	"time"

	"github.com/marcus-crane/gunslinger/config"
	"github.com/marcus-crane/gunslinger/db"
	"github.com/marcus-crane/gunslinger/playback"
)

const (
	spotifyAccessTokenID = "spotify:accesstoken"
	traktAccessTokenID   = "trakt:accesstoken"
)

type IntegrationStatus struct {
	Name         string
	Configured   bool
	HasOAuth     bool
	TokenExpiry  *time.Time
	TokenExpired bool
}

type DebugPageData struct {
	Integrations  []IntegrationStatus
	PlaybackState []playback.FullPlaybackEntry
	Token         string
}

type Handler struct {
	cfg   config.Config
	ps    *playback.PlaybackSystem
	store db.Store
	tmpl  *template.Template
}

func NewHandler(cfg config.Config, ps *playback.PlaybackSystem, store db.Store) *Handler {
	tmpl := template.Must(template.New("debug").Parse(pageTmpl))
	return &Handler{cfg: cfg, ps: ps, store: store, tmpl: tmpl}
}

func (h *Handler) authorized(r *http.Request) bool {
	token := h.cfg.Gunslinger.SuperSecretToken
	return token != "" && r.URL.Query().Get("token") == token
}

func (h *Handler) ServeDebugPage(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := h.buildPageData(r.URL.Query().Get("token"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.Execute(w, data)
}

func (h *Handler) buildPageData(token string) DebugPageData {
	cfg := h.cfg

	integrations := []IntegrationStatus{
		h.oauthStatus("Spotify", cfg.Spotify.ClientId != "", spotifyAccessTokenID),
		h.oauthStatus("Trakt", cfg.Trakt.ClientId != "", traktAccessTokenID),
		{Name: "Plex", Configured: cfg.Plex.Token != ""},
		{Name: "Steam", Configured: cfg.Steam.Token != ""},
		{Name: "RetroAchievements", Configured: cfg.RetroAchievements.Token != ""},
		{Name: "AniList", Configured: cfg.Anilist.Token != ""},
	}

	return DebugPageData{
		Integrations:  integrations,
		PlaybackState: h.ps.State,
		Token:         token,
	}
}

func (h *Handler) oauthStatus(name string, configured bool, tokenID string) IntegrationStatus {
	s := IntegrationStatus{Name: name, Configured: configured, HasOAuth: true}
	if !configured {
		return s
	}
	meta := h.store.GetTokenMetadataByID(tokenID)
	if meta.CreatedAt == 0 {
		return s
	}
	expiry := time.Unix(meta.CreatedAt, 0).Add(time.Duration(meta.ExpiresIn) * time.Second)
	s.TokenExpiry = &expiry
	s.TokenExpired = time.Now().After(expiry)
	return s
}

const pageTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Gunslinger Debug</title>
<style>
  body { font-family: monospace; max-width: 900px; margin: 2rem auto; padding: 0 1rem; }
  h1 { font-size: 1.4rem; }
  h2 { font-size: 1.1rem; margin-top: 2rem; }
  table { border-collapse: collapse; width: 100%; }
  th, td { text-align: left; padding: 0.4rem 0.8rem; border: 1px solid #ccc; }
  th { background: #f4f4f4; }
  .ok   { color: green; }
  .warn { color: orange; }
  .bad  { color: red; }
  .dim  { color: #999; }
</style>
</head>
<body>
<h1>Gunslinger Debug</h1>

<h2>Integrations</h2>
<table>
  <thead>
    <tr>
      <th>Integration</th>
      <th>Configured</th>
      <th>Token Expires</th>
    </tr>
  </thead>
  <tbody>
    {{range .Integrations}}
    <tr>
      <td>{{.Name}}</td>
      <td>{{if .Configured}}<span class="ok">yes</span>{{else}}<span class="bad">no</span>{{end}}</td>
      <td>
        {{if .HasOAuth}}
          {{if .TokenExpiry}}
            {{if .TokenExpired}}
              <span class="bad">expired {{.TokenExpiry.Format "2006-01-02 15:04 MST"}}</span>
            {{else}}
              <span class="ok">{{.TokenExpiry.Format "2006-01-02 15:04 MST"}}</span>
            {{end}}
          {{else}}
            <span class="dim">no token stored</span>
          {{end}}
        {{else}}
          <span class="dim">n/a</span>
        {{end}}
      </td>
    </tr>
    {{end}}
  </tbody>
</table>

<h2>Current Playback</h2>
{{if .PlaybackState}}
<table>
  <thead>
    <tr>
      <th>Source</th>
      <th>Title</th>
      <th>Subtitle</th>
      <th>Category</th>
      <th>Status</th>
      <th>Elapsed</th>
    </tr>
  </thead>
  <tbody>
    {{range .PlaybackState}}
    <tr>
      <td>{{.Source}}</td>
      <td>{{.Title}}</td>
      <td>{{.Subtitle}}</td>
      <td>{{.Category}}</td>
      <td>{{.Status}}</td>
      <td>{{.Elapsed}}</td>
    </tr>
    {{end}}
  </tbody>
</table>
{{else}}
<p class="dim">Nothing currently playing.</p>
{{end}}

</body>
</html>`
