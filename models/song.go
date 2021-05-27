package models

import "gorm.io/gorm"

type Song struct {
	gorm.Model
	AlbumCover string `json:"album_cover"`
	AlbumName  string `json:"album_name"`
	AlbumLink  string `json:"album_link"`
	ArtistName string `json:"artist_name"`
	ArtistLink string `json:"artist_link"`
	Duration   int    `json:"duration"`
	NowPlaying bool   `json:"now_playing"`
	Progress   int    `json:"progress"`
	SongName   string `json:"song_name"`
	SongLink   string `json:"song_link"`
}
