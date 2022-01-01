package models

type Media struct {
	MediaType string  `json:"type"`
	Movie     Movie   `json:"movie"`
	Episode   Episode `json:"episode"`
	Show      Show    `json:"show"`
	StartedAt string  `json:"started_at"`
	ExpiresAt string  `json:"expires_at"`
}

type MediaID struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	TVDB  int    `json:"tvdb"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}

type Movie struct {
	Title    string   `json:"title"`
	Year     int      `json:"year"`
	Link     string   `json:"link"`
	IDs      MediaID  `json:"ids"`
	Poster   []Image  `json:"movie_posters"`
	Tagline  string   `json:"tagline"`
	Overview string   `json:"overview"`
	Language string   `json:"language"`
	Country  string   `json:"country"`
	Runtime  int      `json:"runtime"`
	Genres   []string `json:"genres"`
	Rating   string   `json:"certification"`
}

type Show struct {
	Title     string   `json:"title"`
	Year      int      `json:"year"`
	Link      string   `json:"link"`
	IDs       MediaID  `json:"ids"`
	Backdrops []Image  `json:"show_backdrops"`
	Overview  string   `json:"overview"`
	Runtime   int      `json:"runtime"`
	Rating    int      `json:"rating"`
	Country   string   `json:"country"`
	Network   string   `json:"network"`
	Language  string   `json:"language"`
	Genres    []string `json:"genres"`
}

type Episode struct {
	Title         string  `json:"title"`
	SeasonNumber  int     `json:"season"`
	EpisodeNumber int     `json:"number"`
	Link          string  `json:"link"`
	IDs           MediaID `json:"ids"`
	EpisodeStills []Image `json:"episode_stills"`
	SeasonPosters []Image `json:"season_posters"`
	Overview      string  `json:"overview"`
	Aired         string  `json:"first_aired"`
	Runtime       int     `json:"runtime"`
}

type Backdrops struct {
	Backdrops []Image `json:"backdrops"`
}

type Posters struct {
	Posters []Image `json:"posters"`
}

type Stills struct {
	Stills []Image `json:"stills"`
}

type Image struct {
	AspectRatio float64 `json:"aspect_ratio"`
	Height      int     `json:"height"`
	FilePath    string  `json:"file_path"`
	Width       int     `json:"width"`
}
