package models

type ResponseHTTP struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}
