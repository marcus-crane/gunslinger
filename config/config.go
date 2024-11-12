package config

import (
	"log/slog"
	"strings"
)

type Config struct {
	Anilist           AnilistConfig
	Gunslinger        GunslingerConfig
	Miniflux          MinifluxConfig
	Plex              PlexConfig
	Pushover          PushoverConfig
	Readwise          ReadwiseConfig
	RetroAchievements RetroAchievementsConfig
	Spotify           SpotifyConfig
	Steam             SteamConfig
	Trakt             TraktConfig
}

type AnilistConfig struct {
	Token string `env:"ANILIST_TOKEN"`
}

type GunslingerConfig struct {
	BackgroundJobsEnabled bool   `env:"BACKGROUND_JOBS_ENABLED"`
	DbPath                string `env:"DB_PATH"`
	LogLevel              string `env:"LOG_LEVEL"`
	StorageDir            string `env:"STORAGE_DIR"`
	SuperSecretToken      string `env:"SUPER_SECRET_TOKEN"`
}

type MinifluxConfig struct {
	WebhookSecret string `env:"MINIFLUX_WEBHOOK_SECRET"`
}

type PlexConfig struct {
	Token string `env:"PLEX_TOKEN"`
	URL   string `env:"PLEX_URL"`
}

type PushoverConfig struct {
	Recipient string `env:"PUSHOVER_RECIPIENT"`
	Token     string `env:"PUSHOVER_TOKEN"`
}

type ReadwiseConfig struct {
	Token string `env:"READWISE_TOKEN"`
}

type RetroAchievementsConfig struct {
	Username string `env:"RETROACHIEVEMENTS_USERNAME"`
	Token    string `env:"RETROACHIEVEMENTS_TOKEN"`
}

type SpotifyConfig struct {
	ConnectPlayerName string `env:"SPOTIFY_CONNECT_PLAYER_NAME"`
	ClientId          string `env:"SPOTIFY_CLIENT_ID"`
	ClientSecret      string `env:"SPOTIFY_CLIENT_SECRET"`
	DeviceId          string `env:"SPOTIFY_DEVICE_ID"`
	RedirectUri       string `env:"SPOTIFY_REDIRECT_URI"`
	Username          string `env:"SPOTIFY_USERNAME"`
}

type SteamConfig struct {
	Token string `env:"STEAM_TOKEN"`
}

type TraktConfig struct {
	BearerToken string `env:"TRAKT_BEARER_TOKEN"`
	ClientId    string `env:"TRAKT_CLIENT_ID"`
	TMDBToken   string `env:"TMDB_TOKEN"`
}

func (c *Config) GetLogLevel() slog.Leveler {
	logLevel := strings.ToLower(c.Gunslinger.LogLevel)
	if logLevel == "error" {
		return slog.LevelError
	}
	if logLevel == "warning" {
		return slog.LevelWarn
	}
	if logLevel == "info" {
		return slog.LevelInfo
	}
	if logLevel == "debug" {
		return slog.LevelDebug
	}
	// default to info if unknown
	slog.With(slog.String("log_level", logLevel)).Info("Received invalid log level. Defaulting to INFO.")
	return slog.LevelInfo
}
