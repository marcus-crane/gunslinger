package models

type Media struct {
	MediaType string  `json:"type"`
	Movie     Movie   `json:"movie"`
	Episode   Episode `json:"episode"`
	Show      Show    `json:"show"`
	StartedAt string  `json:"started_at"`
}

type MediaID struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	TVDB  int    `json:"tvdb"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}

type Movie struct {
	Title  string  `json:"title"`
	Year   int     `json:"year"`
	Link   string  `json:"link"`
	IDs    MediaID `json:"ids"`
	Poster Image   `json:"image"`
}

type Show struct {
	Title  string  `json:"title"`
	Year   int     `json:"year"`
	Link   string  `json:"link"`
	IDs    MediaID `json:"ids"`
	Poster Image   `json:"poster"`
}

type Episode struct {
	SeasonNumber  int     `json:"season"`
	EpisodeNumber int     `json:"number"`
	Link          string  `json:"link"`
	IDs           MediaID `json:"ids"`
	EpisodeStill  Image   `json:"episode_still"`
	SeasonPoster  Image   `json:"season_poster"`
}

type Image struct {
	AspectRatio float64 `json:"aspect_ratio"`
	Height      int     `json:"height"`
	FilePath    string  `json:"file_path"`
	Width       int     `json:"width"`
}
