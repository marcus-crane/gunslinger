package obsidian

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ObsidianWebClipperRequest struct {
	Model    string                      `json:"model"`
	Messages []ObsidianWebClipperMessage `json:"messages"`
}

type ObsidianWebClipperMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ObsidianWebClipperResponse struct {
	PromptsResponses map[string]string `json:"prompts_responses"`
}

type OpenAICompatResponse struct {
	Choices []OpenAICompatChoice `json:"choices"`
}

type OpenAICompatChoice struct {
	Message OpenAICompatMessage `json:"message"`
}

type OpenAICompatMessage struct {
	Content string `json:"content"`
}

var (
	ObsidianKagiPromptPlaceholder = "kagi_summarizer_output"
)

func ExtractURLFromWebClipperRequest(r *http.Request) (string, error) {
	var webClipperReq ObsidianWebClipperRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, &webClipperReq)
	if err != nil {
		return "", err
	}

	if len(webClipperReq.Messages) != 3 {
		return "", fmt.Errorf("request was expected to have 3 messages")
	}

	return webClipperReq.Messages[1].Content, nil
}

func FormatWebClipperResponse(summary string) OpenAICompatResponse {
	clipperResp := ObsidianWebClipperResponse{
		PromptsResponses: map[string]string{
			"prompt_1": summary,
		},
	}
	b, _ := json.Marshal(clipperResp)
	openAIResp := OpenAICompatResponse{
		Choices: []OpenAICompatChoice{
			{
				Message: OpenAICompatMessage{
					Content: string(b),
				},
			},
		},
	}
	return openAIResp
}
