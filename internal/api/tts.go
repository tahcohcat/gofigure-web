package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/tahcohcat/gofigure-web/internal/game"
	"github.com/tahcohcat/gofigure-web/internal/tts"
)

type TTSHandler struct {
	ttsClient   tts.Tts
	gameHandler *GameHandler // Reference to access mystery data
}

type TTSRequest struct {
	Text      string `json:"text"`
	Character string `json:"character"`
	Emotion   string `json:"emotion"`
	SessionID string `json:"session_id"` // To get mystery-specific TTS config
}

func NewTTSHandler(gameHandler *GameHandler) (*TTSHandler, error) {
	//cfg, err := config.Load()
	//if err != nil {
	//	return nil, err
	//}

	// Create TTS client (will use Google if configured, dummy otherwise)
	ttsClient, err := tts.NewWebGoogleTTS()
	if err != nil {
		return nil, err
	}

	return &TTSHandler{
		ttsClient:   ttsClient,
		gameHandler: gameHandler,
	}, nil
}

// POST /api/v1/tts/speak - Generate and stream TTS audio
func (th *TTSHandler) SpeakText(w http.ResponseWriter, r *http.Request) {
	var req TTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Character == "" {
		req.Character = "Narrator"
	}
	if req.Emotion == "" {
		req.Emotion = "neutral"
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find TTS model from mystery JSON data
	ttsModel := th.findTTSModelFromMystery(req.SessionID, req.Character)

	// Convert game.TTS to tts.TTSModel
	ttsModelConverted := tts.TTSModel{
		Engine: ttsModel.Engine,
		Model:  ttsModel.Model,
	}
	
	// Generate TTS audio data
	webTTS, ok := th.ttsClient.(tts.WebTTS)
	if !ok {
		http.Error(w, "TTS client doesn't support audio generation", http.StatusInternalServerError)
		return
	}

	audioData, err := webTTS.GenerateAudio(ctx, req.Text, req.Emotion, ttsModelConverted)
	if err != nil {
		http.Error(w, "Failed to generate TTS: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Stream MP3 audio to browser
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Write audio data directly to response
	_, err = w.Write(audioData)
	if err != nil {
		http.Error(w, "Failed to stream audio", http.StatusInternalServerError)
		return
	}
}

// GET /api/v1/tts/test - Test TTS functionality
func (th *TTSHandler) TestTTS(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testText := "Hello, detective. This is a test of the high-quality Google Chirp HD text-to-speech system."
	ttsModel := tts.TTSModel{Engine: "google", Model: "en-US-Chirp-HD-F"}

	// Generate TTS audio data just like the speak endpoint
	webTTS, ok := th.ttsClient.(tts.WebTTS)
	if !ok {
		http.Error(w, "TTS client doesn't support audio generation", http.StatusInternalServerError)
		return
	}

	audioData, err := webTTS.GenerateAudio(ctx, testText, "friendly", ttsModel)
	if err != nil {
		http.Error(w, "TTS test failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Stream MP3 audio to browser
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Write audio data directly to response
	_, err = w.Write(audioData)
	if err != nil {
		http.Error(w, "Failed to stream test audio", http.StatusInternalServerError)
		return
	}
}

// Find TTS model configuration from mystery JSON data
func (th *TTSHandler) findTTSModelFromMystery(sessionID, characterName string) game.TTS {
	// Default high-quality fallback
	defaultModel := game.TTS{Engine: "google", Model: "en-US-Chirp-HD-F"}

	// If no session ID, use default
	if sessionID == "" {
		return defaultModel
	}

	// Get mystery data from game handler
	murder, exists := th.gameHandler.murders[sessionID]
	if !exists {
		return defaultModel
	}

	// Handle narrator
	if characterName == "Narrator" {
		if len(murder.NarratorTTS) > 0 {
			narratorTTS := murder.NarratorTTS[0] // Use first narrator TTS config
			return game.TTS{Engine: narratorTTS.Engine, Model: narratorTTS.Model}
		}
		return defaultModel
	}

	// Find character-specific TTS configuration
	for _, character := range murder.Characters {
		if character.Name == characterName {
			if len(character.TTS) > 0 {
				charTTS := character.TTS[0] // Use first TTS config for character
				return game.TTS{Engine: charTTS.Engine, Model: charTTS.Model}
			}
			break
		}
	}

	return defaultModel
}

func RegisterTTSRoutes(r *mux.Router, gameHandler *GameHandler) {
	ttsHandler, err := NewTTSHandler(gameHandler)
	if err != nil {
		// If TTS setup fails, we'll skip TTS routes
		// This allows the app to run without TTS if Google credentials aren't configured
		return
	}

	r.HandleFunc("/tts/speak", ttsHandler.SpeakText).Methods("POST")
	r.HandleFunc("/tts/test", ttsHandler.TestTTS).Methods("GET")
}
