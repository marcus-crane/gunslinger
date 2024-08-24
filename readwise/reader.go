package readwise

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/marcus-crane/gunslinger/config"
)

type ReaderCount struct {
	New     int `json:"new"`
	Later   int `json:"later"`
	Archive int `json:"archive"`
}

type DocumentList struct {
	Count int `json:"count"`
}

const (
	DocumentListURL = "https://readwise.io/api/v3/list"
)

func GetDocumentCounts(cfg config.Config) (ReaderCount, error) {
	var readerCount ReaderCount
	newCount, err := getDocumentCount(cfg, "new")
	if err != nil {
		return readerCount, err
	}
	laterCount, err := getDocumentCount(cfg, "later")
	if err != nil {
		return readerCount, err
	}
	archiveCount, err := getDocumentCount(cfg, "archive")
	if err != nil {
		return readerCount, err
	}
	readerCount.New = newCount
	readerCount.Later = laterCount
	readerCount.Archive = archiveCount
	return readerCount, nil
}

func getDocumentCount(cfg config.Config, category string) (int, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/?location=%s", DocumentListURL, category), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", cfg.Readwise.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var documentList DocumentList

	err = json.Unmarshal(body, &documentList)
	if err != nil {
		return 0, err
	}
	return documentList.Count, nil
}
