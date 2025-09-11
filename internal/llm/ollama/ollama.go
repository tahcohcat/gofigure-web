package ollama

import (
	"context"
	"fmt"
	"github.com/tahcohcat/gofigure-web/config"
	"github.com/tahcohcat/gofigure-web/internal/logger"
	"time"

	"github.com/ollama/ollama/api"
)

type Client struct {
	client *api.Client
	config *config.OllamaConfig
	logger *logger.Log
}

func NewClient(cfg *config.OllamaConfig) (*Client, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	return &Client{
		client: client,
		config: cfg,
		logger: logger.New(),
	}, nil
}

func (c *Client) GenerateResponse(ctx context.Context, prompt string) (string, error) {

	shouldStream := false

	req := &api.GenerateRequest{
		Model:  c.config.Model,
		Prompt: prompt,
		Stream: &shouldStream,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"top_p":       0.9,
		},
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
	defer cancel()

	c.logger.Debug(fmt.Sprintf("Generating response with model %s", c.config.Model))

	var response string

	// todo: there are lots of interesting
	//  metadata to show in debugging mode
	f := func(g api.GenerateResponse) error {
		response = g.Response
		return nil
	}

	err := c.client.Generate(timeoutCtx, req, f)
	if err != nil {
		c.logger.WithError(err).Error("Failed to generate response")
		return "", fmt.Errorf("ollama generation failed: %w", err)
	}

	return response, nil
}

func (c *Client) IsModelAvailable(ctx context.Context) error {
	models, err := c.client.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	for _, model := range models.Models {
		if model.Name == c.config.Model {
			return nil
		}
	}

	return fmt.Errorf("model %s not found. Available models: %v", c.config.Model, getModelNames(models.Models))
}

func getModelNames(models []api.ListModelResponse) []string {
	names := make([]string, len(models))
	for i, model := range models {
		names[i] = model.Name
	}
	return names
}
