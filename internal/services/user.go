// internal/services/user.go
package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tahcohcat/gofigure-web/internal/database"
	"github.com/tahcohcat/gofigure-web/internal/models"
)

type UserService struct {
	db *database.DB
}

func NewUserService(db *database.DB) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user account
func (s *UserService) CreateUser(req *models.CreateUserRequest) (*models.User, error) {
	// Check if username or email already exists
	if exists, err := s.UsernameExists(req.Username); err != nil {
		return nil, err
	} else if exists {
		return nil, fmt.Errorf("username already exists")
	}

	if exists, err := s.EmailExists(req.Email); err != nil {
		return nil, err
	} else if exists {
		return nil, fmt.Errorf("email already exists")
	}

	user := &models.User{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := user.SetPassword(req.Password); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert user into database
	query := `
		INSERT INTO users (username, email, password_hash, display_name, created_at, updated_at, is_active)
		VALUES (:username, :email, :password_hash, :display_name, :created_at, :updated_at, :is_active)
	`

	result, err := s.db.NamedExec(query, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	user.ID = int(id)

	// Initialize user stats
	if err := s.initializeUserStats(user.ID); err != nil {
		// Non-fatal error, just log it
		fmt.Printf("Warning: failed to initialize user stats for user %d: %v\n", user.ID, err)
	}

	return user, nil
}

// AuthenticateUser validates login credentials and returns the user
func (s *UserService) AuthenticateUser(req *models.LoginRequest) (*models.User, error) {
	user, err := s.GetUserByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.CheckPassword(req.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// Update last login time
	if err := s.UpdateLastLogin(user.ID); err != nil {
		// Non-fatal error, just log it
		fmt.Printf("Warning: failed to update last login for user %d: %v\n", user.ID, err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (s *UserService) GetUserByID(id int) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, display_name, created_at, updated_at, last_login_at, is_active 
			  FROM users WHERE id = ?`

	err := s.db.Get(&user, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by their username
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, password_hash, display_name, created_at, updated_at, last_login_at, is_active 
			  FROM users WHERE username = ?`

	err := s.db.Get(&user, query, username)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// UsernameExists checks if a username is already taken
func (s *UserService) UsernameExists(username string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE username = ?`
	err := s.db.Get(&count, query, username)
	return count > 0, err
}

// EmailExists checks if an email is already registered
func (s *UserService) EmailExists(email string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE email = ?`
	err := s.db.Get(&count, query, email)
	return count > 0, err
}

// UpdateLastLogin updates the user's last login timestamp
func (s *UserService) UpdateLastLogin(userID int) error {
	query := `UPDATE users SET last_login_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), userID)
	return err
}

// GetUserStats retrieves user gameplay statistics
func (s *UserService) GetUserStats(userID int) (*models.UserStats, error) {
	var stats models.UserStats
	query := `SELECT user_id, games_played, games_won, total_play_time, fastest_solve, favorite_mystery 
			  FROM user_stats WHERE user_id = ?`

	err := s.db.Get(&stats, query, userID)
	if err == sql.ErrNoRows {
		// Initialize stats if they don't exist
		return &models.UserStats{
			UserID:          userID,
			GamesPlayed:     0,
			GamesWon:        0,
			TotalPlayTime:   0,
			FastestSolve:    0,
			FavoriteMystery: "",
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return &stats, nil
}

// CreateGameSession records the start of a new game session
func (s *UserService) CreateGameSession(userID int, mysteryID, sessionID string) error {
	query := `
		INSERT INTO user_game_sessions (user_id, mystery_id, session_id, started_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, userID, mysteryID, sessionID, time.Now())
	return err
}

// CompleteGameSession records the completion of a game session
func (s *UserService) CompleteGameSession(sessionID string, solved bool, timeSpent, questionsAsked int) error {
	query := `
		UPDATE user_game_sessions 
		SET finished_at = ?, solved = ?, time_spent = ?, questions_asked = ?
		WHERE session_id = ?
	`
	_, err := s.db.Exec(query, time.Now(), solved, timeSpent, questionsAsked, sessionID)
	return err
}

// initializeUserStats creates initial stats record for a new user
func (s *UserService) initializeUserStats(userID int) error {
	query := `
		INSERT OR IGNORE INTO user_stats (user_id, games_played, games_won, total_play_time, fastest_solve, favorite_mystery)
		VALUES (?, 0, 0, 0, 0, '')
	`
	_, err := s.db.Exec(query, userID)
	return err
}

// UpdateProfile allows users to update their display name and email
func (s *UserService) UpdateProfile(userID int, displayName, email string) error {
	// Check if email is taken by another user
	var count int
	query := `SELECT COUNT(*) FROM users WHERE email = ? AND id != ?`
	if err := s.db.Get(&count, query, email, userID); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("email already exists")
	}

	query = `UPDATE users SET display_name = ?, email = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, displayName, email, time.Now(), userID)
	return err
}

// ChangePassword allows users to change their password
func (s *UserService) ChangePassword(userID int, currentPassword, newPassword string) error {
	// Get user to verify current password
	var user models.User
	query := `SELECT password_hash FROM users WHERE id = ?`
	if err := s.db.Get(&user, query, userID); err != nil {
		return fmt.Errorf("user not found")
	}

	if !user.CheckPassword(currentPassword) {
		return fmt.Errorf("current password is incorrect")
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update in database
	updateQuery := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(updateQuery, user.Password, time.Now(), userID)
	return err
}
