package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/tahcohcat/gofigure-web/internal/game"
)

type GameHandler struct {
	murders   map[string]*game.Murder  // Store loaded mysteries by ID
	engine    *game.WebEngine         // Game engine instance
}

func NewGameHandler() *GameHandler {
	engine, err := game.NewWebEngine()
	if err != nil {
		panic("Failed to create web engine: " + err.Error())
	}

	return &GameHandler{
		murders: make(map[string]*game.Murder),
		engine:  engine,
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

	// Store the game session (in production, use proper session management)
	sessionID := generateSessionID()
	gh.murders[sessionID] = &murder

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
		"title":      murder.Title,
		"intro":      murder.Intro,
		"characters": murder.Characters,
		"killer":     murder.Killer,    // Include killer info for accusation checking
		"location":   murder.Location,
		"weapon":     murder.Weapon,
	})
}

// POST /api/v1/game/{session}/ask - Ask a character a question
func (gh *GameHandler) AskCharacter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session"]

	murder, exists := gh.murders[sessionID]
	if !exists {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}

	var req struct {
		CharacterName string `json:"character_name"`
		Question      string `json:"question"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find the character
	var character *game.Character
	for i := range murder.Characters {
		if murder.Characters[i].Name == req.CharacterName {
			character = &murder.Characters[i]
			break
		}
	}

	if character == nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	// Use the game engine to get character response
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reply, err := gh.engine.AskCharacterQuestion(ctx, character, req.Question, *murder)
	if err != nil {
		http.Error(w, "Failed to get character response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"character": req.CharacterName,
		"question":  req.Question,
		"response":  reply.Response,
		"emotion":   reply.Emotion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func RegisterRoutes(r *mux.Router) *GameHandler {
	gh := NewGameHandler()
	
	r.HandleFunc("/mysteries", gh.ListMysteries).Methods("GET")
	r.HandleFunc("/game/start", gh.StartGame).Methods("POST")
	r.HandleFunc("/game/{session}/ask", gh.AskCharacter).Methods("POST")
	
	return gh
}

// Simple session ID generator (use UUID in production)
func generateSessionID() string {
	return "session_123" // Placeholder
}