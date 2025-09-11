package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	LLM    LLMConfig    `mapstructure:"llm"`
	Ollama OllamaConfig `mapstructure:"ollama"`
	OpenAI OpenAIConfig `mapstructure:"openai"`
	Tts    TtsConfig    `mapstructure:"tts"`
	Sst    SstConfig    `mapstructure:"sst"`
}

// LLM provider selection
type LLMConfig struct {
	Provider string `mapstructure:"provider"` // "ollama" or "openai"
}

// New OpenAI config
type OpenAIConfig struct {
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"model"`
	BaseURL   string `mapstructure:"base_url"`   // Optional, defaults to OpenAI API
	MaxTokens int    `mapstructure:"max_tokens"` // Optional, defaults to model's max
	Timeout   int    `mapstructure:"timeout"`
}

type TtsConfig struct {
	Type    string `mapstructure:"type"`
	Enabled bool   `mapstructure:"enabled"`
}

type SstConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Provider     string `mapstructure:"provider"`
	LanguageCode string `mapstructure:"language_code"`
	SampleRate   int    `mapstructure:"sample_rate"`
}

type OllamaConfig struct {
	Host    string `mapstructure:"host"`
	Model   string `mapstructure:"model"`
	Timeout int    `mapstructure:"timeout"` // seconds
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.BindEnv("openai.api_key", "GOFIGURE_OPENAI_API_KEY")
	viper.BindEnv("openai.model", "OPENAI_MODEL")
	viper.BindEnv("openai.base_url", "OPENAI_BASE_URL")
	viper.BindEnv("llm.provider", "LLM_PROVIDER")

	viper.SetDefault("ollama.host", "http://localhost:11434")
	viper.SetDefault("ollama.model", "llama3.2")
	viper.SetDefault("ollama.timeout", 50)

	viper.SetDefault("openai.timeout", 30)
	viper.SetDefault("openai.max_tokens", 1000)
	viper.SetDefault("ollama.timeout", 30)

	viper.SetDefault("llm.provider", "openai")

	viper.SetDefault("tts.enabled", true)
	viper.SetDefault("tts.type", "google")

	viper.SetDefault("sst.enabled", true)
	viper.SetDefault("sst.provider", "google")
	viper.SetDefault("sst.language_code", "en-US")
	viper.SetDefault("sst.sample_rate", 16000)

	// Allow environment variables
	viper.SetEnvPrefix("GOFIGURE")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found, use defaults
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
