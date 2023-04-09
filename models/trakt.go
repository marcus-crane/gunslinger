package models

type NowPlayingResponse struct {
	ExpiresAt string       `json:"expires_at"`
	StartedAt string       `json:"started_at"`
	Action    string       `json:"action"`
	Type      string       `json:"type"`
	Movie     TraktSummary `json:"movie"`
	Episode   TraktEpisode `json:"episode"`
	Show      TraktSummary `json:"show"`
}

type TraktEpisode struct {
	Season int      `json:"season"`
	Number int      `json:"number"`
	Title  string   `json:"title"`
	IDs    TraktIDs `json:"ids"`
}

type TraktSummary struct {
	Title string   `json:"title"`
	Year  int      `json:"year"`
	IDs   TraktIDs `json:"ids"`
}

type TraktIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	TVDB  int    `json:"tvdb"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}

type TMDBImageResponse struct {
	ID      int         `json:"id"`
	Stills  []TMDBImage `json:"stills"`
	Posters []TMDBImage `json:"posters"`
}

type TMDBImage struct {
	FilePath string `json:"file_path"`
}

// {
// 	"id": 3282633,
// 	"stills": [
// 		{
// 			"aspect_ratio": 1.778,
// 			"height": 1080,
// 			"iso_639_1": null,
// 			"file_path": "/mtJIGRYjuSR1NGwE9VSfYqwHH6s.jpg",
// 			"vote_average": 0.0,
// 			"vote_count": 0,
// 			"width": 1920
// 		},
// 		{
// 			"aspect_ratio": 1.778,
// 			"height": 1080,
// 			"iso_639_1": null,
// 			"file_path": "/yWvdab9mkEipzMkBVtOw4d9LFMl.jpg",
// 			"vote_average": 0.0,
// 			"vote_count": 0,
// 			"width": 1920
// 		}
// 	]
// }
