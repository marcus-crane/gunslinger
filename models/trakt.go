package models

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
