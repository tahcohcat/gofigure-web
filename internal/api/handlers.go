// internal/api/handlers.go (Updated with user integration)
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
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tahcohcat/gofigure-web/internal/auth"
	"github.com/tahcohcat/gofigure-web/internal/game"
	"github.com/tahcohcat/gofigure-web/internal/logger"
	"github.com/tahcohcat/gofigure-web/internal/models"
	"github.com/tahcohcat/gofigure-web/internal/services"
)

type GameSession struct {
	UserID         int
	Murder         *game.Murder
	Timer          *time.Ticker
	RemainingTime  int
	TimerEnabled   bool
	GameOver       bool
	StartedAt      time.Time
	QuestionsAsked int
}

type GameHandler struct {
	sessions           map[string]*GameSession // Store game sessions by ID
	engine             *game.WebEngine         // Game engine instance
	userService        *services.UserService   // User service for database operations
	achievementService *services.AchievementService
}

func NewGameHandler(userService *services.UserService) *GameHandler {
	engine, err := game.NewWebEngine()
	if err != nil {
		panic("Failed to create web engine: " + err.Error())
	}

	achievementService := services.NewAchievementService(userService.GetDB())

	return &GameHandler{
		sessions:           make(map[string]*GameSession),
		engine:             engine,
		userService:        userService,
		achievementService: achievementService,
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
	// Get user ID from session
	userID := auth.GetUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

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
		UserID:         userID,
		Murder:         &murder,
		RemainingTime:  3600, // 1 hour
		TimerEnabled:   true,
		GameOver:       false,
		StartedAt:      time.Now(),
		QuestionsAsked: 0,
	}
	gh.sessions[sessionID] = session

	// Record game session start in database
	if err := gh.userService.CreateGameSession(userID, req.MysteryID, sessionID); err != nil {
		log.Printf("Warning: failed to record game session start: %v", err)
	}

	// Start the game timer
	session.Timer = time.NewTicker(1 * time.Second)
	go func() {
		for range session.Timer.C {
			if session.TimerEnabled && !session.GameOver {
				session.RemainingTime--
				if session.RemainingTime <= 0 {
					session.GameOver = true
					session.Timer.Stop()

					// Auto-complete the game session as unsolved when time runs out
					timeSpent := int(time.Since(session.StartedAt).Seconds())
					if err := gh.userService.CompleteGameSession(sessionID, false, timeSpent, session.QuestionsAsked); err != nil {
						log.Printf("Warning: failed to complete game session on timeout: %v", err)
					}
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

	// Verify user owns this session
	userID := auth.GetUserIDFromSession(r)
	if session.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
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

	// Increment questions asked counter
	session.QuestionsAsked++

	// Record question activity for achievements
	achievementData := map[string]interface{}{
		"character":       req.CharacterName,
		"question":        req.Question,
		"total_questions": session.QuestionsAsked,
	}

	if err := gh.achievementService.CheckAndUpdateAchievements(userID, "question_asked", achievementData); err != nil {
		log.Printf("Warning: failed to check question achievements: %v", err)
	}

	// Calculate stress response
	newStressLevel, stressState := calculateStressResponse(req.Question, character, req.CurrentStress)
	stressChange := newStressLevel - req.CurrentStress

	// Log the interaction for debugging
	log.Printf("User %d - Character %s stress: %.1f -> %.1f (change: +%.1f) - State: %s",
		userID, character.Name, req.CurrentStress, newStressLevel, stressChange, stressState)

	logger.New().Info(fmt.Sprintf("User %d - Character %s stress: %.1f -> %.1f (change: +%.1f) - State: %s",
		userID, character.Name, req.CurrentStress, newStressLevel, stressChange, stressState))

	// Use the game engine to get character response
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

// POST /api/v1/game/{session}/accuse - Make an accusation
func (gh *GameHandler) MakeAccusation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session"]

	session, exists := gh.sessions[sessionID]
	if !exists {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}

	// Verify user owns this session
	userID := auth.GetUserIDFromSession(r)
	if session.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if session.GameOver {
		http.Error(w, "Game is already over", http.StatusBadRequest)
		return
	}

	var req struct {
		Suspect string `json:"suspect"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if the accusation is correct
	correct := req.Suspect == session.Murder.Killer

	// Mark game as over
	session.GameOver = true
	if session.Timer != nil {
		session.Timer.Stop()
	}

	// Calculate time spent
	timeSpent := int(time.Since(session.StartedAt).Seconds())

	// Record game completion in database
	if err := gh.userService.CompleteGameSession(sessionID, correct, timeSpent, session.QuestionsAsked); err != nil {
		log.Printf("Warning: failed to complete game session: %v", err)
	}

	response := map[string]interface{}{
		"correct":    correct,
		"killer":     session.Murder.Killer,
		"weapon":     session.Murder.Weapon,
		"location":   session.Murder.Location,
		"time_spent": timeSpent,
		"questions":  session.QuestionsAsked,
	}

	// Record stats
	// Record game completion in database
	if err := gh.userService.CompleteGameSession(sessionID, correct, timeSpent, session.QuestionsAsked); err != nil {
		log.Printf("Warning: failed to complete game session: %v", err)
	}

	// Record activity
	mysteryTitle := session.Murder.Title
	if correct {
		gh.achievementService.RecordActivity(userID, "mystery_solved",
			fmt.Sprintf("Solved \"%s\" mystery", mysteryTitle),
			fmt.Sprintf("Time: %d:%02d, Questions: %d", timeSpent/60, timeSpent%60, session.QuestionsAsked),
			"🎯")

		// Check for new personal record
		userStats, _ := gh.userService.GetUserStats(userID)
		if userStats.FastestSolve == 0 || timeSpent < userStats.FastestSolve {
			gh.achievementService.RecordActivity(userID, "record_set",
				fmt.Sprintf("New personal record: %d:%02d", timeSpent/60, timeSpent%60),
				"", "⚡")
		}
	} else {
		gh.achievementService.RecordActivity(userID, "mystery_attempted",
			fmt.Sprintf("Attempted \"%s\" mystery", mysteryTitle),
			fmt.Sprintf("Time: %d:%02d, Questions: %d", timeSpent/60, timeSpent%60, session.QuestionsAsked),
			"🎯")
	}

	// Check and update achievements
	achievementData := map[string]interface{}{
		"time_spent":      timeSpent,
		"questions_asked": session.QuestionsAsked,
		"mystery_id":      sessionID, // You might want to store mystery ID in session
		"correct":         correct,
	}

	if err := gh.achievementService.CheckAndUpdateAchievements(userID, "mystery_solved", achievementData); err != nil {
		log.Printf("Warning: failed to check achievements: %v", err)
	}

	if correct {
		response["message"] = fmt.Sprintf("🎉 Congratulations! You correctly identified %s as the killer!", session.Murder.Killer)
	} else {
		response["message"] = fmt.Sprintf("❌ Sorry, that's incorrect. The real killer was %s.", session.Murder.Killer)
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

	// Verify user owns this session
	userID := auth.GetUserIDFromSession(r)
	if session.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
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

	// Verify user owns this session
	userID := auth.GetUserIDFromSession(r)
	if session.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	session.TimerEnabled = !session.TimerEnabled

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"timer_enabled": session.TimerEnabled,
	})
}

// GET /api/v1/profile/stats - Get user stats (alternative endpoint)
func (gh *GameHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	stats, err := gh.userService.GetUserStats(userID)
	if err != nil {
		http.Error(w, "Failed to get user stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GET /api/v1/profile/achievements - Get user achievements
func (gh *GameHandler) GetUserAchievements(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	achievements, err := gh.achievementService.GetUserAchievements(userID)
	if err != nil {
		log.Printf("Failed to get user achievements: %v", err)
		http.Error(w, "Failed to get achievements", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"achievements": achievements,
	})
}

// GET /api/v1/profile/activities - Get recent user activities
func (gh *GameHandler) GetUserActivities(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	activities, err := gh.achievementService.GetRecentActivities(userID, limit)
	if err != nil {
		log.Printf("Failed to get user activities: %v", err)
		http.Error(w, "Failed to get activities", http.StatusInternalServerError)
		return
	}

	// Format activities for frontend
	formattedActivities := make([]map[string]interface{}, len(activities))
	for i, activity := range activities {
		timeAgo := formatTimeAgo(activity.CreatedAt)
		formattedActivities[i] = map[string]interface{}{
			"id":         activity.ID,
			"type":       activity.Type,
			"title":      activity.Title,
			"details":    activity.Details,
			"icon":       activity.Icon,
			"time":       timeAgo,
			"created_at": activity.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activities": formattedActivities,
	})
}

// GET /api/v1/profile/full - Get complete user profile with stats, achievements, and activities
func (gh *GameHandler) GetFullUserProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get user info
	user, err := gh.userService.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get user stats
	stats, err := gh.userService.GetUserStats(userID)
	if err != nil {
		log.Printf("Failed to get user stats: %v", err)
		http.Error(w, "Failed to get user stats", http.StatusInternalServerError)
		return
	}

	// Get achievements
	achievements, err := gh.achievementService.GetUserAchievements(userID)
	if err != nil {
		log.Printf("Failed to get user achievements: %v", err)
		// Don't fail the whole request, just return empty achievements
		achievements = []models.UserAchievementView{}
	}

	// Get recent activities
	activities, err := gh.achievementService.GetRecentActivities(userID, 10)
	if err != nil {
		log.Printf("Failed to get user activities: %v", err)
		activities = []models.GameActivity{}
	}

	// Calculate derived stats
	successRate := 0
	if stats.GamesPlayed > 0 {
		successRate = int(float64(stats.GamesWon) / float64(stats.GamesPlayed) * 100)
	}

	badgesEarned := 0
	for _, achievement := range achievements {
		if achievement.Completed {
			badgesEarned++
		}
	}

	// Format fastest solve time
	fastestSolve := "N/A"
	if stats.FastestSolve > 0 {
		minutes := stats.FastestSolve / 60
		seconds := stats.FastestSolve % 60
		fastestSolve = fmt.Sprintf("%d:%02d", minutes, seconds)
	}

	// Format total play time
	totalPlayTimeHours := float64(stats.TotalPlayTime) / 3600.0

	// Determine detective rank based on achievements and stats
	detectiveRank := determineDetectiveRank(badgesEarned, stats.GamesWon, successRate)

	// Format activities
	formattedActivities := make([]map[string]interface{}, len(activities))
	for i, activity := range activities {
		timeAgo := formatTimeAgo(activity.CreatedAt)
		formattedActivities[i] = map[string]interface{}{
			"icon": activity.Icon,
			"text": activity.Title,
			"time": timeAgo,
		}
	}

	response := map[string]interface{}{
		"user": map[string]interface{}{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"created_at":   user.CreatedAt,
		},
		"stats": map[string]interface{}{
			"games_played":    stats.GamesPlayed,
			"games_won":       stats.GamesWon,
			"success_rate":    successRate,
			"fastest_solve":   fastestSolve,
			"total_play_time": totalPlayTimeHours,
			"badges_earned":   badgesEarned,
			"detective_rank":  detectiveRank,
		},
		"achievements": achievements,
		"activities":   formattedActivities,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to determine detective rank
func determineDetectiveRank(badgesEarned, gamesWon, successRate int) string {
	if badgesEarned >= 10 && successRate >= 80 {
		return "🕵️ Master Detective"
	} else if badgesEarned >= 6 && gamesWon >= 15 {
		return "🎖️ Senior Detective"
	} else if badgesEarned >= 3 && gamesWon >= 5 {
		return "👮 Detective"
	} else if gamesWon >= 1 {
		return "🔍 Junior Detective"
	}
	return "👤 Detective Trainee"
}

// Helper function to format time ago
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

func RegisterRoutes(r *mux.Router, userService *services.UserService) *GameHandler {
	gh := NewGameHandler(userService)

	r.HandleFunc("/mysteries", gh.ListMysteries).Methods("GET")
	r.HandleFunc("/game/start", gh.StartGame).Methods("POST")
	r.HandleFunc("/game/{session}/ask", gh.AskCharacter).Methods("POST")
	r.HandleFunc("/game/{session}/accuse", gh.MakeAccusation).Methods("POST")
	r.HandleFunc("/game/{session}/timer", gh.GetTimer).Methods("GET")
	r.HandleFunc("/game/{session}/timer/toggle", gh.ToggleTimer).Methods("POST")
	r.HandleFunc("/profile/stats", gh.GetUserStats).Methods("GET")

	r.HandleFunc("/profile/full", gh.GetFullUserProfile).Methods("GET")
	r.HandleFunc("/profile/achievements", gh.GetUserAchievements).Methods("GET")
	r.HandleFunc("/profile/activities", gh.GetUserActivities).Methods("GET")

	return gh
}

// Simple session ID generator (use UUID in production)
func generateSessionID() string {
	return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
}

func GetUserProfile(userService *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetUserIDFromSession(r)
		if userID == 0 {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		user, err := userService.GetUserByID(userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		stats, err := userService.GetUserStats(userID)
		if err != nil {
			http.Error(w, "Failed to get user stats", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user":  user,
			"stats": stats,
		})
	}
}

func UpdateUserProfile(userService *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetUserIDFromSession(r)
		if userID == 0 {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		var req models.ProfileUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := userService.UpdateProfile(userID, req.DisplayName, req.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func ChangePassword(userService *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetUserIDFromSession(r)
		if userID == 0 {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		var req models.PasswordChangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := userService.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
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
