package kagi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/marcus-crane/gunslinger/config"
)

type KagiSummarizerResponse struct {
	Data KagiSummarizerData `json:"data"`
}

type KagiSummarizerData struct {
	Output string `json:"output"`
}

var (
	KagiSummarizerURL = "https://kagi.com/api/v0/summarize"
)

func SummarizeURL(cfg config.Config, summaryUrl string) (string, error) {
	kagiUrl := fmt.Sprintf("%s?url=%s&summary_type=takeaway", KagiSummarizerURL, url.QueryEscape(summaryUrl))
	req, err := http.NewRequest("POST", kagiUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bot %s", cfg.Kagi.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	fmt.Printf("status %d\n", resp.StatusCode)
	if resp.StatusCode != 200 {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var kagiResp KagiSummarizerResponse
	err = json.Unmarshal(body, &kagiResp)
	if err != nil {
		return "", err
	}
	return kagiResp.Data.Output, nil
}
