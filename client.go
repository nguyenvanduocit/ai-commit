package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
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

type GptClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

const chatEndpoint = "https://api.openai.com/v1/chat/completions"

func NewGptClient(apiKey string, model string) *GptClient {
	return &GptClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: http.DefaultClient,
	}
}

func (c *GptClient) ChatComplete(ctx context.Context, messages []*Message) (string, error) {

	request := &ChatCompleteRequest{
		Model:    c.model,
		Messages: messages,
	}

	payload, _ := json.Marshal(request)
	payloadReader := bytes.NewReader(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", chatEndpoint, payloadReader)
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.apiKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	// check for errors
	if res.StatusCode != http.StatusOK {
		errorMessage := gjson.GetBytes(body, "error.message").String()
		if errorMessage == "" {
			errorMessage = string(body)
		}

		return "", errors.New("failed to get response from OpenAI API: " + errorMessage)
	}

	answer := gjson.GetBytes(body, "choices.0.message.content").String()
	answer = strings.TrimSpace(answer)
	answer = strings.Trim(answer, `"`)

	return answer, nil

}

// SingleQuestion asks a single question to the user
func (c *GptClient) SingleQuestion(question string) (string, error) {
	message := []*Message{
		{
			Role:    "user",
			Content: question,
		},
	}

	response, err := c.ChatComplete(context.Background(), message)
	if err != nil {
		return "", err
	}

	return response, nil
}
