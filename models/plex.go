package models

type PlexResponse struct {
	MediaContainer MediaContainer `json:"MediaContainer"`
}

type MediaContainer struct {
	Size     int        `json:"size"`
	Metadata []Metadata `json:"Metadata"`
}

type Metadata struct {
	Attribution      string     `json:"attribution"`
	Duration         int        `json:"duration"`
	GrandparentTitle string     `json:"grandparentTitle"`
	Thumb            string     `json:"thumb"`
	ParentThumb      string     `json:"parentThumb"`
	Index            int        `json:"index"`
	ParentIndex      int        `json:"parentIndex"`
	Title            string     `json:"title"`
	Type             string     `json:"type"`
	ViewOffset       int        `json:"viewOffset"`
	Director         []Director `json:"Director"`
	Player           Player     `json:"Player"`
}

type Director struct {
	Name string `json:"tag"`
}

type Player struct {
	State string `json:"state"`
}
