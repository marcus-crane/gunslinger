package readwise

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/marcus-crane/gunslinger/utils"
)

const (
	BooksURL = "https://readwise.io/api/v2/books/?page_size=1000"
)

var (
	ReadwiseToken = utils.MustEnv("READWISE_TOKEN")
)

type BookList struct {
	Count    int      `json:"count"`
	Next     string   `json:"next"`
	Previous string   `json:"previous"`
	Results  []Result `json:"results"`
}

type Result struct {
	Category      string `json:"category"`
	NumHighlights int    `json:"num_highlights"`
	Tags          []Tag  `json:"tags"`
}

type Tag struct {
	Name string `json:"name"`
}

type TagResult struct {
	ArticlesProcessed     int `json:"articles_processed"`
	ArticlesUnprocessedBM int `json:"articles_unprocessed_bm"`
	ArticlesTotal         int `json:"articles_total"`
	BooksProcessed        int `json:"books_processed"`
	BooksUnprocessedBM    int `json:"books_unprocessed_bm"`
	BooksTotal            int `json:"books_total"`
	PodcastsProcessed     int `json:"podcasts_processed"`
	PodcastsTotal         int `json:"podcasts_total"`
	TweetsProcessed       int `json:"tweets_processed"`
	TweetsTotal           int `json:"tweets_total"`
}

func CountTags() (TagResult, error) {
	var tagResult TagResult
	req, err := http.NewRequest("GET", BooksURL, nil)
	if err != nil {
		return tagResult, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", ReadwiseToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tagResult, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tagResult, err
	}

	var bookList BookList

	err = json.Unmarshal(body, &bookList)
	if err != nil {
		return tagResult, err
	}

	for _, res := range bookList.Results {
		if res.NumHighlights == 0 {
			continue
		}
		if res.Category == "articles" {
			tagResult.ArticlesTotal += 1
		}
		if res.Category == "books" {
			tagResult.BooksTotal += 1
		}
		if res.Category == "podcasts" {
			tagResult.PodcastsTotal += 1
		}
		if res.Category == "tweets" {
			tagResult.TweetsTotal += 1
		}
		for _, tag := range res.Tags {
			if tag.Name == "meta/processed" {
				if res.Category == "articles" {
					tagResult.ArticlesProcessed += 1
				}
				if res.Category == "books" {
					tagResult.BooksProcessed += 1
				}
				if res.Category == "podcasts" {
					tagResult.PodcastsProcessed += 1
				}
				if res.Category == "tweets" {
					tagResult.TweetsProcessed += 1
				}
			}
		}
	}

	tagResult.ArticlesUnprocessedBM = 450 - tagResult.ArticlesProcessed
	tagResult.BooksUnprocessedBM = 56 - tagResult.BooksProcessed
	return tagResult, nil
}
