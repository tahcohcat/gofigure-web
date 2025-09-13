package models

import (
	"database/sql/driver"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user account
type User struct {
	ID          int        `json:"id" db:"id"`
	Username    string     `json:"username" db:"username"`
	Email       string     `json:"email" db:"email"`
	Password    string     `json:"-" db:"password_hash"` // Never expose in JSON
	DisplayName string     `json:"display_name" db:"display_name"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at" db:"last_login_at"`
	IsActive    bool       `json:"is_active" db:"is_active"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=20"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=50"`
}

// LoginRequest represents a login attempt
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserStats represents user gameplay statistics
type UserStats struct {
	UserID          int    `json:"user_id" db:"user_id"`
	    GamesPlayed     int    `json:"games_played" db:"games_played"`
	GamesWon        int    `json:"games_won" db:"games_won"`
	TotalPlayTime   int    `json:"total_play_time" db:"total_play_time"` // in seconds
	FastestSolve    int    `json:"fastest_solve" db:"fastest_solve"`     // in seconds, 0 = no solves
	FavoriteMystery string `json:"favorite_mystery" db:"favorite_mystery"`
}

// UserGameSession represents a game session record
type UserGameSession struct {
	ID             int        `json:"id" db:"id"`
	UserID         int        `json:"user_id" db:"user_id"`
	MysteryID      string     `json:"mystery_id" db:"mystery_id"`
	SessionID      string     `json:"session_id" db:"session_id"`
	StartedAt      time.Time  `json:"started_at" db:"started_at"`
	FinishedAt     *time.Time `json:"finished_at" db:"finished_at"`
	Solved         *bool      `json:"solved" db:"solved"`
	TimeSpent      *int       `json:"time_spent" db:"time_spent"` // in seconds
	QuestionsAsked *int       `json:"questions_asked" db:"questions_asked"`
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies a password against the user's hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// Value implements the driver.Valuer interface for database storage
func (u *User) Value() (driver.Value, error) {
	return u.ID, nil
}

// ProfileUpdateRequest represents a profile update request
type ProfileUpdateRequest struct {
	DisplayName string `json:"display_name" validate:"required,min=1,max=50"`
	Email       string `json:"email" validate:"required,email"`
}

// PasswordChangeRequest represents a password change request
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}
