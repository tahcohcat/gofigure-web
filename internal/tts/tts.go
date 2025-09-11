package tts

import "context"

// TTSModel represents TTS configuration from mystery JSON
type TTSModel struct {
	Engine string `json:"engine"`
	Model  string `json:"model"`
}

type Tts interface {
	Speak(ctx context.Context, text, emotion string, model TTSModel) error
	Name() string
}

// WebTTS interface for generating audio data instead of playing
type WebTTS interface {
	Tts
	GenerateAudio(ctx context.Context, text, emotion string, model TTSModel) ([]byte, error)
}

// Factory function for creating TTS clients
func NewWebGoogleTTS() (Tts, error) {
	return NewWebGoogleTTSClient()
}
