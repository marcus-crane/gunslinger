package models

type SteamPlayerSummary struct {
	Response SteamResponse `json:"response"`
}

type SteamResponse struct {
	Players []SteamUser `json:"players"`
}

type SteamUser struct {
	GameID       string `json:"gameid"`
	PersonaState int    `json:"personastate"`
}

type SteamAppResponse struct {
	Data SteamAppDetail `json:"data"`
}

type SteamAppDetail struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	HeaderImage string   `json:"header_image"`
	Developers  []string `json:"developers"`
}
