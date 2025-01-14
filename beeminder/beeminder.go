package beeminder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/marcus-crane/gunslinger/config"
)

const (
	datapointsFmtUrl = "https://www.beeminder.com/api/v1/users/utf9k/goals/%s/datapoints.json"
	goalsUrl         = "https://www.beeminder.com/api/v1/users/utf9k/goals.json"
)

type Goal struct {
	Slug            string   `json:"slug"`
	Title           string   `json:"title"`
	Rate            float32  `json:"rate"`
	GraphUrl        string   `json:"graph_url"`
	ThumbUrl        string   `json:"thumb_url"`
	SafeSum         string   `json:"safesum"`
	BareMin         string   `json:"baremin"`
	GUnits          string   `json:"gunits"`
	Contract        Contract `json:"contract"`
	RoadStatusColor string   `json:"roadstatuscolor"`
	HeadSum         string   `json:"headsum"`
}

type Contract struct {
	Amount float32 `json:"amount"`
}

func FetchGoals(cfg config.Config) ([]Goal, error) {
	var goals []Goal
	resp, err := http.Get(fmt.Sprintf("%s?auth_token=%s", goalsUrl, cfg.Beeminder.Token))
	if err != nil {
		return goals, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return goals, err
	}

	err = json.Unmarshal(body, &goals)
	if err != nil {
		return goals, err
	}
	return goals, nil
}

func SubmitDatapoint(cfg config.Config, goalName string, value string, comment string) error {
	url := fmt.Sprintf(datapointsFmtUrl, goalName)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Form.Add("value", value)
	req.Form.Add("comment", fmt.Sprintf("[Gunslinger] %s", comment))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		return nil
	}
	return err
}
