package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tahcohcat/gofigure-web/internal/database"
	"github.com/tahcohcat/gofigure-web/internal/models"
)

type AchievementService struct {
	db *database.DB
}

func NewAchievementService(db *database.DB) *AchievementService {
	return &AchievementService{db: db}
}

// GetUserAchievements returns all achievements with user's progress
func (s *AchievementService) GetUserAchievements(userID int) ([]models.UserAchievementView, error) {
	query := `
		SELECT 
			a.id, a.icon, a.title, a.description, a.type, a.category, a.max_progress, a.created_at,
			COALESCE(ua.progress, 0) as progress,
			COALESCE(ua.completed, false) as completed,
			ua.completed_at
		FROM achievements a
		LEFT JOIN user_achievements ua ON a.id = ua.achievement_id AND ua.user_id = ?
		ORDER BY ua.completed DESC, a.category, a.created_at
	`

	var achievements []models.UserAchievementView
	err := s.db.Select(&achievements, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user achievements: %w", err)
	}

	return achievements, nil
}

// UpdateAchievementProgress updates or creates user achievement progress
func (s *AchievementService) UpdateAchievementProgress(userID int, achievementID string, progress int) error {
	// First check if achievement exists and get max progress
	var achievement models.Achievement
	err := s.db.Get(&achievement, "SELECT * FROM achievements WHERE id = ?", achievementID)
	if err != nil {
		return fmt.Errorf("achievement not found: %w", err)
	}

	// Cap progress at max_progress
	if achievement.MaxProgress > 0 && progress > achievement.MaxProgress {
		progress = achievement.MaxProgress
	}

	completed := false
	var completedAt *time.Time

	// Check if achievement is completed
	if achievement.MaxProgress == 0 {
		// Binary achievement (no progress tracking)
		completed = progress > 0
	} else {
		// Progress-based achievement
		completed = progress >= achievement.MaxProgress
	}

	if completed {
		now := time.Now()
		completedAt = &now
	}

	// Upsert user achievement
	query := `
		INSERT INTO user_achievements (user_id, achievement_id, progress, completed, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, achievement_id) DO UPDATE SET
			progress = ?,
			completed = ?,
			completed_at = CASE WHEN ? THEN ? ELSE completed_at END,
			updated_at = ?
	`

	now := time.Now()
	_, err = s.db.Exec(query,
		userID, achievementID, progress, completed, completedAt, now, now,
		progress, completed, completed, completedAt, now)

	if err != nil {
		return fmt.Errorf("failed to update achievement progress: %w", err)
	}

	// If achievement was just completed, record activity
	if completed && completedAt != nil {
		s.RecordActivity(userID, "badge_earned", fmt.Sprintf("Earned \"%s\" badge", achievement.Title), "", achievement.Icon)
	}

	return nil
}

// CheckAndUpdateAchievements checks various achievement conditions after game events
func (s *AchievementService) CheckAndUpdateAchievements(userID int, event string, data map[string]interface{}) error {
	switch event {
	case "mystery_solved":
		return s.checkMysteryAchievements(userID, data)
	case "question_asked":
		return s.checkQuestionAchievements(userID, data)
	case "game_started":
		return s.checkGameStartAchievements(userID, data)
	}
	return nil
}

func (s *AchievementService) checkMysteryAchievements(userID int, data map[string]interface{}) error {
	// Get user stats for checking achievements
	stats, err := s.getUserStatsForAchievements(userID)
	if err != nil {
		return err
	}

	// First Case achievement
	if stats.GamesWon == 1 {
		s.UpdateAchievementProgress(userID, "first-case", 1)
	}

	// Speed Demon (solve in under 15 minutes)
	if timeSpent, ok := data["time_spent"].(int); ok {
		if timeSpent < 900 { // 15 minutes
			s.UpdateAchievementProgress(userID, "speed-demon", 1)
		}
	}

	// Efficient Detective (solve with < 20 questions)
	if questionsAsked, ok := data["questions_asked"].(int); ok {
		if questionsAsked < 20 {
			s.UpdateAchievementProgress(userID, "efficient", 1)
		}
	}

	// Perfect Ten (10 mysteries in a row)
	consecutiveWins := s.getConsecutiveWins(userID)
	if consecutiveWins >= 10 {
		s.UpdateAchievementProgress(userID, "perfect-ten", 10)
	} else {
		s.UpdateAchievementProgress(userID, "perfect-ten", consecutiveWins)
	}

	// Night Owl (solve after midnight)
	solveTime := time.Now()
	if solveTime.Hour() >= 0 && solveTime.Hour() < 6 {
		s.UpdateAchievementProgress(userID, "night-owl", 1)
	}

	// Weekend Warrior (solve 5 mysteries on weekends)
	if solveTime.Weekday() == time.Saturday || solveTime.Weekday() == time.Sunday {
		// Get current weekend warrior progress and increment
		current := s.getAchievementProgress(userID, "weekend-warrior")
		s.UpdateAchievementProgress(userID, "weekend-warrior", current+1)
	}

	// Mystery Maven (solve all available mysteries)
	totalMysteries := 4 // Update this based on your available mysteries
	if stats.GamesWon >= totalMysteries {
		s.UpdateAchievementProgress(userID, "mystery-maven", stats.GamesWon)
	}

	// Veteran Detective (play for 30 days)
	daysSinceFirstGame := s.getDaysSinceFirstGame(userID)
	if daysSinceFirstGame >= 30 {
		s.UpdateAchievementProgress(userID, "veteran", 1)
	}

	// Sherlock Holmes (90% success rate with 20+ cases)
	if stats.GamesPlayed >= 20 {
		successRate := float64(stats.GamesWon) / float64(stats.GamesPlayed) * 100
		if successRate >= 90 {
			s.UpdateAchievementProgress(userID, "sherlock", 1)
		} else {
			s.UpdateAchievementProgress(userID, "sherlock-progress", int(successRate))
		}
	}

	return nil
}

func (s *AchievementService) checkQuestionAchievements(userID int, data map[string]interface{}) error {
	// The Interrogator (ask 100 questions total)
	totalQuestions := s.getTotalQuestionsAsked(userID)
	s.UpdateAchievementProgress(userID, "interrogator", totalQuestions)

	return nil
}

func (s *AchievementService) checkGameStartAchievements(userID int, data map[string]interface{}) error {
	// Social Butterfly (talk to every character in a mystery)
	// This would be checked when the game ends based on character interaction data
	return nil
}

// Helper methods
func (s *AchievementService) getUserStatsForAchievements(userID int) (*models.UserStats, error) {
	var stats models.UserStats
	query := `SELECT user_id, games_played, games_won, total_play_time, fastest_solve, favorite_mystery 
			  FROM user_stats WHERE user_id = ?`

	err := s.db.Get(&stats, query, userID)
	if err == sql.ErrNoRows {
		return &models.UserStats{UserID: userID}, nil
	}
	return &stats, err
}

func (s *AchievementService) getConsecutiveWins(userID int) int {
	query := `
		WITH recent_games AS (
			SELECT solved, ROW_NUMBER() OVER (ORDER BY finished_at DESC) as rn
			FROM user_game_sessions 
			WHERE user_id = ? AND finished_at IS NOT NULL
			ORDER BY finished_at DESC
			LIMIT 20
		)
		SELECT COUNT(*) as consecutive_wins
		FROM recent_games 
		WHERE solved = true AND rn <= (
			SELECT COALESCE(MIN(rn), 21) 
			FROM recent_games 
			WHERE solved = false
		) - 1
	`

	var consecutiveWins int
	err := s.db.Get(&consecutiveWins, query, userID)
	if err != nil {
		return 0
	}
	return consecutiveWins
}

func (s *AchievementService) getAchievementProgress(userID int, achievementID string) int {
	var progress int
	query := `SELECT COALESCE(progress, 0) FROM user_achievements WHERE user_id = ? AND achievement_id = ?`
	err := s.db.Get(&progress, query, userID, achievementID)
	if err != nil {
		return 0
	}
	return progress
}

func (s *AchievementService) getTotalQuestionsAsked(userID int) int {
	var total int
	query := `SELECT COALESCE(SUM(questions_asked), 0) FROM user_game_sessions WHERE user_id = ?`
	err := s.db.Get(&total, query, userID)
	if err != nil {
		return 0
	}
	return total
}

func (s *AchievementService) getDaysSinceFirstGame(userID int) int {
	var firstGame time.Time
	query := `SELECT MIN(started_at) FROM user_game_sessions WHERE user_id = ?`
	err := s.db.Get(&firstGame, query, userID)
	if err != nil {
		return 0
	}
	return int(time.Since(firstGame).Hours() / 24)
}

// RecordActivity adds a new activity entry for the user
func (s *AchievementService) RecordActivity(userID int, activityType, title, details, icon string) error {
	query := `
		INSERT INTO game_activities (user_id, type, title, details, icon, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, userID, activityType, title, details, icon, time.Now())
	return err
}

// GetRecentActivities returns recent user activities
func (s *AchievementService) GetRecentActivities(userID int, limit int) ([]models.GameActivity, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, user_id, type, title, details, icon, created_at
		FROM game_activities 
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	var activities []models.GameActivity
	err := s.db.Select(&activities, query, userID, limit)
	return activities, err
}

// Seed default achievements
func (s *AchievementService) SeedDefaultAchievements() error {
	achievements := []models.Achievement{
		{ID: "first-case", Icon: "🎯", Title: "First Case", Description: "Solve your first mystery", Type: "milestone", Category: "progress"},
		{ID: "speed-demon", Icon: "⚡", Title: "Speed Demon", Description: "Solve a mystery in under 15 minutes", Type: "challenge", Category: "time"},
		{ID: "interrogator", Icon: "🗣️", Title: "The Interrogator", Description: "Ask 100 questions across all mysteries", Type: "progress", Category: "questions", MaxProgress: 100},
		{ID: "perfect-ten", Icon: "💯", Title: "Perfect Ten", Description: "Solve 10 mysteries in a row", Type: "progress", Category: "streak", MaxProgress: 10},
		{ID: "night-owl", Icon: "🌙", Title: "Night Owl Detective", Description: "Solve a mystery after midnight", Type: "special", Category: "time"},
		{ID: "efficient", Icon: "🎪", Title: "Efficient Detective", Description: "Solve a mystery with less than 20 questions", Type: "challenge", Category: "efficiency"},
		{ID: "social-butterfly", Icon: "👥", Title: "Social Butterfly", Description: "Talk to every character in a mystery", Type: "progress", Category: "social", MaxProgress: 5},
		{ID: "weekend-warrior", Icon: "🏖️", Title: "Weekend Warrior", Description: "Solve 5 mysteries on weekends", Type: "progress", Category: "time", MaxProgress: 5},
		{ID: "mystery-maven", Icon: "🔍", Title: "Mystery Maven", Description: "Solve all available mysteries", Type: "collection", Category: "completion", MaxProgress: 4},
		{ID: "comeback-king", Icon: "👑", Title: "Comeback King", Description: "Solve a mystery after 3 wrong accusations", Type: "special", Category: "resilience"},
		{ID: "veteran", Icon: "⭐", Title: "Veteran Detective", Description: "Play for 30 days", Type: "milestone", Category: "loyalty"},
		{ID: "sherlock", Icon: "🎩", Title: "Sherlock Holmes", Description: "Achieve 90% success rate with 20+ cases", Type: "mastery", Category: "skill"},
	}

	for _, achievement := range achievements {
		query := `
			INSERT OR IGNORE INTO achievements (id, icon, title, description, type, category, max_progress, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := s.db.Exec(query, achievement.ID, achievement.Icon, achievement.Title,
			achievement.Description, achievement.Type, achievement.Category, achievement.MaxProgress, time.Now())
		if err != nil {
			return fmt.Errorf("failed to seed achievement %s: %w", achievement.ID, err)
		}
	}

	return nil
}
