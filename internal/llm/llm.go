// internal/llm/interface.go
package llm

import (
	"context"
)

type CharacterReply struct {
	Response string `json:"response"`
	Emotion  string `json:"emotion"`
}

// LLM defines the interface for language model providers
type LLM interface {

	// GenerateResponse generates a response from the LLM given a prompt
	GenerateResponse(ctx context.Context, prompt string) (string, error)

	// IsModelAvailable checks if the configured model is available
	IsModelAvailable(ctx context.Context) error
}
