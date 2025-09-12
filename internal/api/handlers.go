package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tahcohcat/gofigure-web/internal/game"
	"github.com/tahcohcat/gofigure-web/internal/logger"
)

type GameSession struct {
	Murder        *game.Murder
	Timer         *time.Ticker
	RemainingTime int
	TimerEnabled  bool
	GameOver      bool
}

type GameHandler struct {
	sessions map[string]*GameSession // Store game sessions by ID
	engine   *game.WebEngine        // Game engine instance
}

func NewGameHandler() *GameHandler {
	engine, err := game.NewWebEngine()
	if err != nil {
		panic("Failed to create web engine: " + err.Error())
	}

	return &GameHandler{
		sessions: make(map[string]*GameSession),
		engine:   engine,
	}
}

// GET /api/v1/mysteries - List available mysteries
func (gh *GameHandler) ListMysteries(w http.ResponseWriter, r *http.Request) {
	mysteries := []map[string]interface{}{
		{
			"id":          "diner_secrets",
			"title":       "Secrets at Rosie's Diner",
			"description": "A small-town mystery where everyone has secrets",
			"difficulty":  "Easy",
			"file":        "data/mysteries/diner_secrets.json",
		},
		{
			"id":          "blackwood",
			"title":       "The Blackwood Manor Murder",
			"description": "A classic manor house mystery with a stormy night setting",
			"difficulty":  "Medium",
			"file":        "data/mysteries/blackwood.json",
		},
		{
			"id":          "corporate_betrayal",
			"title":       "Corporate Betrayal",
			"description": "A modern office murder involving corporate secrets and embezzlement",
			"difficulty":  "Medium",
			"file":        "data/mysteries/corporate_betrayal.json",
		},
		{
			"id":          "cruise_ship",
			"title":       "Death on the Aurora Star",
			"description": "A luxury cruise ship mystery with complex motives and alibis",
			"difficulty":  "Hard",
			"file":        "data/mysteries/cruise_ship.json",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mysteries": mysteries,
	})
}

// POST /api/v1/game/start - Start a new game with a mystery
func (gh *GameHandler) StartGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MysteryID string `json:"mystery_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Load the mystery file
	mysteryFile := filepath.Join("data/mysteries", req.MysteryID+".json")
	murder, err := game.LoadMurderFromFile(mysteryFile)
	if err != nil {
		http.Error(w, "Failed to load mystery: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create and store the game session
	sessionID := generateSessionID()
	session := &GameSession{
		Murder:        &murder,
		RemainingTime: 3600, // 1 hour
		TimerEnabled:  true,
		GameOver:      false,
	}
	gh.sessions[sessionID] = session

	// Start the game timer
	session.Timer = time.NewTicker(1 * time.Second)
	go func() {
		for range session.Timer.C {
			if session.TimerEnabled && !session.GameOver {
				session.RemainingTime--
				if session.RemainingTime <= 0 {
					session.GameOver = true
					session.Timer.Stop()
				}
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
		"title":      murder.Title,
		"intro":      murder.Intro,
		"characters": murder.Characters,
		"killer":     murder.Killer, // Include killer info for accusation checking
		"location":   murder.Location,
		"weapon":     murder.Weapon,
	})
}

// Add to your ask question request struct
type AskQuestionRequest struct {
	CharacterName string  `json:"character_name"`
	Question      string  `json:"question"`
	CurrentStress float64 `json:"current_stress"`
}

type CharacterResponse struct {
	Character    string  `json:"character"`
	Question     string  `json:"question"`
	Response     string  `json:"response"`
	Emotion      string  `json:"emotion"`
	StressLevel  float64 `json:"stress_level"`
	StressChange float64 `json:"stress_change"`
	StressState  string  `json:"stress_state"`
}

// POST /api/v1/game/{session}/ask - Ask a character a question
func (gh *GameHandler) AskCharacter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session"]

	session, exists := gh.sessions[sessionID]
	if !exists {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}

	if session.GameOver {
		http.Error(w, "Game is over", http.StatusBadRequest)
		return
	}

	var req AskQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find the character
	var character *game.Character
	for i := range session.Murder.Characters {
		if session.Murder.Characters[i].Name == req.CharacterName {
			character = &session.Murder.Characters[i]
			break
		}
	}

	if character == nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	// Calculate stress response
	newStressLevel, stressState := calculateStressResponse(req.Question, character, req.CurrentStress)
	stressChange := newStressLevel - req.CurrentStress

	// Log the interaction for debugging
	log.Printf("Character %s stress: %.1f -> %.1f (change: +%.1f) - State: %s",
		character.Name, req.CurrentStress, newStressLevel, stressChange, stressState)

	logger.New().Info(fmt.Sprintf("Character %s stress: %.1f -> %.1f (change: +%.1f) - State: %s",
		character.Name, req.CurrentStress, newStressLevel, stressChange, stressState))

	// Use the game engine to get character response
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//todo: add stress to ai interaction
	reply, err := gh.engine.AskCharacterQuestion(ctx, character, req.Question, *session.Murder)
	if err != nil {
		http.Error(w, "Failed to get character response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := CharacterResponse{
		Character:    req.CharacterName,
		Question:     req.Question,
		Response:     reply.Response,
		Emotion:      reply.Emotion,
		StressState:  stressState,
		StressChange: stressChange,
		StressLevel:  newStressLevel,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /api/v1/game/{session}/timer - Get remaining time
func (gh *GameHandler) GetTimer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session"]

	session, exists := gh.sessions[sessionID]
	if !exists {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"remaining_time": session.RemainingTime,
		"timer_enabled":  session.TimerEnabled,
		"game_over":      session.GameOver,
	})
}

// POST /api/v1/game/{session}/timer/toggle - Toggle the timer
func (gh *GameHandler) ToggleTimer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session"]

	session, exists := gh.sessions[sessionID]
	if !exists {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}

	session.TimerEnabled = !session.TimerEnabled

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"timer_enabled": session.TimerEnabled,
	})
}

func RegisterRoutes(r *mux.Router) *GameHandler {
	gh := NewGameHandler()

	r.HandleFunc("/mysteries", gh.ListMysteries).Methods("GET")
	r.HandleFunc("/game/start", gh.StartGame).Methods("POST")
	r.HandleFunc("/game/{session}/ask", gh.AskCharacter).Methods("POST")
	r.HandleFunc("/game/{session}/timer", gh.GetTimer).Methods("GET")
	r.HandleFunc("/game/{session}/timer/toggle", gh.ToggleTimer).Methods("POST")

	return gh
}

// Simple session ID generator (use UUID in production)
func generateSessionID() string {
	return "session_123" // Placeholder
}

// Add stress calculation function
func calculateStressResponse(question string, character *game.Character, currentStress float64) (float64, string) {
	questionLower := strings.ToLower(question)
	stressIncrease := 5.0 // Base stress increase

	// High stress keywords
	highStressKeywords := []string{
		"murder", "kill", "weapon", "blood", "death", "guilty",
		"lie", "alibi", "where were you", "motive", "why did you",
	}

	// Medium stress keywords
	mediumStressKeywords := []string{
		"suspicious", "secret", "hidden", "truth", "evidence",
		"witness", "saw", "heard", "relationship", "money",
	}

	// Low stress keywords (calming topics)
	lowStressKeywords := []string{
		"weather", "family", "work", "hobby", "general",
		"hello", "how are", "nice day", "background",
	}

	// Calculate stress based on keywords
	for _, keyword := range highStressKeywords {
		if strings.Contains(questionLower, keyword) {
			stressIncrease += 15.0
		}
	}

	for _, keyword := range mediumStressKeywords {
		if strings.Contains(questionLower, keyword) {
			stressIncrease += 8.0
		}
	}

	for _, keyword := range lowStressKeywords {
		if strings.Contains(questionLower, keyword) {
			stressIncrease = math.Max(1.0, stressIncrease-5.0)
		}
	}

	// Character personality modifiers
	personalityLower := strings.ToLower(character.Personality)
	if strings.Contains(personalityLower, "nervous") {
		stressIncrease *= 1.3
	}
	if strings.Contains(personalityLower, "calm") {
		stressIncrease *= 0.7
	}
	if strings.Contains(personalityLower, "secretive") {
		stressIncrease *= 1.2
	}
	if strings.Contains(personalityLower, "aggressive") {
		stressIncrease *= 1.1
	}

	// Add some randomness
	randomFactor := (rand.Float64() - 0.5) * 10 // ±5 variation
	stressIncrease += randomFactor

	// Calculate new stress level
	newStressLevel := math.Min(100.0, currentStress+stressIncrease)

	// Determine stress state
	var stressState string
	switch {
	case newStressLevel < 25:
		stressState = "calm"
	case newStressLevel < 40:
		stressState = "composed"
	case newStressLevel < 55:
		stressState = "nervous"
	case newStressLevel < 70:
		stressState = "agitated"
	case newStressLevel < 85:
		stressState = "stressed"
	default:
		stressState = "nervous"
	}

	return newStressLevel, stressState
}
