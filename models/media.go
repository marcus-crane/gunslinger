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
	Title string  `json:"title"`
	Year  int     `json:"year"`
	Link  string  `json:"link"`
	IDs   MediaID `json:"ids"`
}

type Show struct {
	Title string  `json:"title"`
	Year  int     `json:"year"`
	Link  string  `json:"link"`
	IDs   MediaID `json:"ids"`
}

type Episode struct {
	SeasonNumber  int     `json:"season"`
	EpisodeNumber int     `json:"number"`
	Link          string  `json:"link"`
	IDs           MediaID `json:"ids"`
}
