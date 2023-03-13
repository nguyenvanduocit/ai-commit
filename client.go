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

	// check for errors
	if res.StatusCode != http.StatusOK {
		errorMessage := gjson.GetBytes(body, "error.message").String()
		if errorMessage == "" {
			errorMessage = string(body)
		}

		return "", errors.New("failed to get response from OpenAI API: " + errorMessage)
	}

	var response ChatCompleteResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", errors.New("no choices returned from OpenAI API")
	}

	firstChoice := response.Choices[0].Message.Content
	firstChoice = strings.TrimSpace(firstChoice)
	firstChoice = strings.Trim(firstChoice, `"`)

	return firstChoice, nil

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
