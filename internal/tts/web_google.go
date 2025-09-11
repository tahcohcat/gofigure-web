package tts

import (
	"context"
	"fmt"
	"os"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	tts "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/tahcohcat/gofigure-web/internal/logger"
)

type WebGoogleTTS struct {
	client *texttospeech.Client
	logger *logger.Log
}

func NewWebGoogleTTSClient() (*WebGoogleTTS, error) {
	ctx := context.Background()

	if jsonCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); jsonCreds != "" {
		client, err := texttospeech.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed creating google tts client for os.json-creds: %w", err)
		}
		return &WebGoogleTTS{client: client, logger: logger.New()}, nil
	}

	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google TTS client: %w", err)
	}

	return &WebGoogleTTS{
		client: client,
		logger: logger.New(),
	}, nil
}

// Extract language code from model name (e.g., "en-US-Chirp-HD-F" -> "en-US", "en-GB-Standard-D" -> "en-GB")
func (g *WebGoogleTTS) extractLanguageCode(modelName string) string {
	parts := strings.Split(modelName, "-")
	if len(parts) >= 2 {
		return fmt.Sprintf("%s-%s", parts[0], parts[1])
	}
	// Fallback to en-US if we can't parse
	return "en-US"
}

// GenerateAudio generates TTS audio data without playing it
func (g *WebGoogleTTS) GenerateAudio(ctx context.Context, text, emotion string, model TTSModel) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Clean up text for TTS
	cleanText := strings.ReplaceAll(text, "[", "")
	cleanText = strings.ReplaceAll(cleanText, "]", "")

	// Extract language code from model name
	languageCode := g.extractLanguageCode(model.Model)

	// Build the synthesis request
	req := &tts.SynthesizeSpeechRequest{
		Input: &tts.SynthesisInput{
			InputSource: &tts.SynthesisInput_Text{Text: cleanText},
		},
		Voice: &tts.VoiceSelectionParams{
			LanguageCode: languageCode, // Dynamic language code based on model
			Name:         model.Model,  // Full model name (e.g., "en-GB-Standard-D")
		},
		AudioConfig: &tts.AudioConfig{
			AudioEncoding:   tts.AudioEncoding_MP3, // Use MP3 for web compatibility
			SpeakingRate:    g.getSpeakingRateForEmotion(emotion),
			Pitch:           g.getPitchForEmotion(emotion),
			VolumeGainDb:    0.0,
			SampleRateHertz: 22050, // Good quality for web
		},
	}

	g.logger.Debug(fmt.Sprintf("Generating Google TTS audio with voice: %s, language: %s, emotion: %s",
		model.Model, languageCode, emotion))

	// Generate the audio
	resp, err := g.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}

	if len(resp.AudioContent) == 0 {
		return nil, fmt.Errorf("empty audio content received from Google TTS")
	}

	g.logger.Debug(fmt.Sprintf("Generated %d bytes of MP3 audio", len(resp.AudioContent)))
	return resp.AudioContent, nil
}

// Speak implementation for compatibility (stores audio for later retrieval)
func (g *WebGoogleTTS) Speak(ctx context.Context, text, emotion string, model TTSModel) error {
	_, err := g.GenerateAudio(ctx, text, emotion, model)
	return err
}

func (g *WebGoogleTTS) Name() string {
	return "Google Cloud Text-to-Speech (Web)"
}

func (g *WebGoogleTTS) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// Helper functions for emotion-based voice modulation
func (g *WebGoogleTTS) getSpeakingRateForEmotion(emotion string) float64 {
	switch strings.ToLower(emotion) {
	case "excited", "happy", "energetic":
		return 1.15
	case "angry", "frustrated":
		return 1.10
	case "nervous", "worried", "anxious":
		return 1.25
	case "sad", "melancholy", "depressed":
		return 0.85
	case "mysterious", "suspicious":
		return 0.90
	case "calm", "peaceful", "serene":
		return 0.95
	default:
		return 1.0
	}
}

func (g *WebGoogleTTS) getPitchForEmotion(emotion string) float64 {
	switch strings.ToLower(emotion) {
	case "excited", "happy", "surprised":
		return 2.0
	case "angry", "frustrated":
		return -2.0
	case "nervous", "worried":
		return 3.0
	case "sad", "melancholy":
		return -3.0
	case "mysterious", "suspicious":
		return -1.5
	case "authoritative", "confident":
		return -1.0
	default:
		return 0.0
	}
}
