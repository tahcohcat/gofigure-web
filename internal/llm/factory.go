// internal/llm/factory.go
package llm

import (
	"fmt"
	"github.com/tahcohcat/gofigure-web/config"
	"github.com/tahcohcat/gofigure-web/internal/llm/ollama"
	"github.com/tahcohcat/gofigure-web/internal/llm/openai"
)

type Provider string

const (
	ProviderOllama Provider = "ollama"
	ProviderOpenAI Provider = "openai"
)

// NewLLMClient creates a new LLM client based on the configuration
func NewLLMClient(cfg *config.Config) (LLM, error) {
	switch Provider(cfg.LLM.Provider) {
	case ProviderOllama:
		return ollama.NewClient(&cfg.Ollama)
	case ProviderOpenAI:
		return openai.NewClient(&cfg.OpenAI)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLM.Provider)
	}
}
