package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompleteRequest struct {
	Model    string     `json:"model"`
	Messages []*Message `json:"messages"`
}

type ChatCompleteChoice struct {
	Index   int `json:"index"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type ChatCompleteResponse struct {
	Id      string               `json:"id"`
	Object  string               `json:"object"`
	Created int                  `json:"created"`
	Choices []ChatCompleteChoice `json:"choices"`
}

type GptClient struct {
	apiKey     string
	httpClient *http.Client
}

const chatModel = "gpt-3.5-turbo"
const chatEndpoint = "https://api.openai.com/v1/chat/completions"

func NewGptClient(apiKey string) *GptClient {
	return &GptClient{
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
	}
}

func (c *GptClient) ChatComplete(ctx context.Context, messages []*Message) (string, error) {

	request := &ChatCompleteRequest{
		Model:    chatModel,
		Messages: messages,
	}

	payload, _ := json.Marshal(request)
	payloadReader := bytes.NewReader(payload)
	req, err := http.NewRequest("POST", chatEndpoint, payloadReader)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response ChatCompleteResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "I don't know", nil
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil

}

var agreeWords = []string{
	"yes",
	"y",
	"ok",
	"okay",
	"agree",
}

// IsAgree returns true if the user agrees with the commit message
func (c *GptClient) IsAgree(userResponse string) bool {
	for _, word := range agreeWords {
		if strings.HasPrefix(strings.ToLower(userResponse), word) {
			return true
		}
	}

	message := []*Message{
		{
			Role:    "user",
			Content: "only response with \"change request\" or \"agreement\"; the following message is a change request or agreement: " + userResponse,
		},
	}

	response, err := c.ChatComplete(context.Background(), message)
	if err != nil {
		return false
	}

	lowerResponse := strings.ToLower(response)

	return strings.HasPrefix(lowerResponse, "agreement")
}
