package models

type AnilistResponse struct {
	Data AnilistData `json:"data"`
}

type AnilistData struct {
	Page Page `json:"Page"`
}

type Page struct {
	Activities []Activity `json:"activities"`
}

type Activity struct {
	Id        int64  `json:"id"`
	Status    string `json:"status"`
	Progress  string `json:"progress"`
	CreatedAt int64  `json:"createdAt"`
	Media     Manga  `json:"media"`
}

type Manga struct {
	Id         int64      `json:"id"`
	Title      MangaTitle `json:"title"`
	Chapters   int        `json:"chapters"`
	CoverImage MangaCover `json:"coverImage"`
}

type MangaTitle struct {
	UserPreferred string `json:"userPreferred"`
}

type MangaCover struct {
	ExtraLarge string `json:"extraLarge"`
}
