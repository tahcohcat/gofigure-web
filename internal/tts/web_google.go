package tts

import (
	"context"
	"fmt"
	"os"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	tts "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/tahcohcat/gofigure-web/internal/logger"
	"google.golang.org/api/option"
)

type WebGoogleTTS struct {
	client *texttospeech.Client
	logger *logger.Log
}

func NewWebGoogleTTSClient() (*WebGoogleTTS, error) {
	ctx := context.Background()
	logger := logger.New()

	jsonCreds, found := os.LookupEnv("GOOGLE_CREDENTIALS_JSON")
	if found {
		if jsonCreds != "" {
			logger.Info("Found GOOGLE_CREDENTIALS_JSON environment variable with content. Initializing client with it.")
			opts := option.WithCredentialsJSON([]byte(jsonCreds))
			client, err := texttospeech.NewClient(ctx, opts)
			if err != nil {
				return nil, fmt.Errorf("failed creating google tts client from json env var: %w", err)
			}
			return &WebGoogleTTS{client: client, logger: logger}, nil
		} else {
			logger.Warn("GOOGLE_CREDENTIALS_JSON environment variable is set but empty. Falling back to default credentials.")
		}
	} else {
		logger.Info("GOOGLE_CREDENTIALS_JSON environment variable not found. Falling back to default credentials.")
	}

	// For other environments, rely on the default credential provider chain.
	// This chain will automatically look for GOOGLE_APPLICATION_CREDENTIALS file,
	// gcloud credentials, and metadata server credentials.
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google TTS client using default credentials: %w", err)
	}

	return &WebGoogleTTS{
		client: client,
		logger: logger,
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
	audioConfig := &tts.AudioConfig{
		AudioEncoding:   tts.AudioEncoding_MP3,
		SpeakingRate:    g.getSpeakingRateForEmotion(emotion),
		VolumeGainDb:    0.0,
		SampleRateHertz: 22050,
	}

	// Conditionally add pitch only if the voice model is not a Chirp voice
	if !strings.Contains(model.Model, "Chirp") {
		audioConfig.Pitch = g.getPitchForEmotion(emotion)
	}

	req := &tts.SynthesizeSpeechRequest{
		Input: &tts.SynthesisInput{
			InputSource: &tts.SynthesisInput_Text{Text: cleanText},
		},
		Voice: &tts.VoiceSelectionParams{
			LanguageCode: languageCode,
			Name:         model.Model,
		},
		AudioConfig: audioConfig,
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
		return 1.20
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
