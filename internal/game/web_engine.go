package game

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tahcohcat/gofigure-web/config"
	llmpkg "github.com/tahcohcat/gofigure-web/internal/llm"
	"github.com/tahcohcat/gofigure-web/internal/logger"
)

// WebEngine is a simplified version of the game engine for web use
type WebEngine struct {
	config *config.Config
	logger *logger.Log
}

func NewWebEngine() (*WebEngine, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &WebEngine{
		config: cfg,
		logger: logger.New(),
	}, nil
}

// LoadMurderFromFile loads a murder mystery from a JSON file
func LoadMurderFromFile(filename string) (Murder, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Murder{}, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var murder Murder
	if err := decoder.Decode(&murder); err != nil {
		return Murder{}, fmt.Errorf("failed to decode mystery JSON: %w", err)
	}

	return murder, nil
}

// AskCharacterQuestion handles character interaction for the web interface
func (e *WebEngine) AskCharacterQuestion(ctx context.Context, character *Character, question string, murder Murder) (*llmpkg.CharacterReply, error) {
	// Create LLM client
	llmClient, err := llmpkg.NewLLMClient(e.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Use the character's AskQuestion method
	reply, err := character.AskQuestion(ctx, question, murder, llmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get character response: %w", err)
	}

	return reply, nil
}