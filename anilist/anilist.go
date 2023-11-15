package anilist

import (
	"net/http"
	"time"
)

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: "https://graphql.anilist.co",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
