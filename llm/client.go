package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	model   string
	apiKey  string
	timeout time.Duration
	client  *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewClient(baseURL, model, apiKey string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Chat(messages []ChatMessage) (string, error) {
	// Check if baseURL already includes the path
	var url string
	if strings.Contains(c.baseURL, "/v1/chat/completions") {
		url = c.baseURL
	} else {
		// Remove trailing slash if present, then append path
		baseURL := strings.TrimSuffix(c.baseURL, "/")
		url = fmt.Sprintf("%s/v1/chat/completions", baseURL)
	}
	
	reqBody := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}
	
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	
	return chatResp.Choices[0].Message.Content, nil
}

func (c *Client) Prompt(prompt string) (string, error) {
	messages := []ChatMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}
	return c.Chat(messages)
}

