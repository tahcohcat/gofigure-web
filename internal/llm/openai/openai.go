// internal/openai/openai.go
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tahcohcat/gofigure-web/config"
	"github.com/tahcohcat/gofigure-web/internal/logger"
	"io"
	"net/http"
	"time"
)

type Client struct {
	apiKey     string
	baseURL    string
	config     *config.OpenAIConfig
	logger     *logger.Log
	httpClient *http.Client
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type ModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

func NewClient(cfg *config.OpenAIConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		config:  cfg,
		logger:  logger.New(),
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}, nil
}

func (c *Client) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// Parse the prompt - assuming it's JSON serialized conversation
	var messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(prompt), &messages); err != nil {
		// If it's not JSON, treat as a simple prompt
		messages = []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: prompt},
		}
	}

	// Convert to OpenAI format
	var openaiMessages []OpenAIMessage
	for _, msg := range messages {
		openaiMessages = append(openaiMessages, OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	req := OpenAIRequest{
		Model:       c.config.Model,
		Messages:    openaiMessages,
		Temperature: 0.7,
		MaxTokens:   c.config.MaxTokens,
		Stream:      false,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	c.logger.Debug(fmt.Sprintf("Generating response with OpenAI model %s", c.config.Model))

	requestBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.WithError(err).Error("Failed to make OpenAI request")
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error(fmt.Sprintf("OpenAI API returned status %d: %s", resp.StatusCode, string(body)))
		return "", fmt.Errorf("openai API error: status %d", resp.StatusCode)
	}

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if openaiResp.Error != nil {
		return "", fmt.Errorf("openai API error: %s", openaiResp.Error.Message)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	response := openaiResp.Choices[0].Message.Content
	c.logger.Debug(fmt.Sprintf("Generated response: %d tokens used", openaiResp.Usage.TotalTokens))

	return response, nil
}

func (c *Client) IsModelAvailable(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to list models: status %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return fmt.Errorf("failed to unmarshal models response: %w", err)
	}

	// Check if the configured model is available
	for _, model := range modelsResp.Data {
		if model.ID == c.config.Model {
			return nil
		}
	}

	// Get list of available models for error message
	var availableModels []string
	for _, model := range modelsResp.Data {
		availableModels = append(availableModels, model.ID)
	}

	return fmt.Errorf("model %s not found. Available models: %v", c.config.Model, availableModels)
}
