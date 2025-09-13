package models

import (
	"time"
)

type Achievement struct {
	ID          string    `json:"id" db:"id"`
	Icon        string    `json:"icon" db:"icon"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Type        string    `json:"type" db:"type"` // milestone, challenge, progress, special, collection, mastery
	Category    string    `json:"category" db:"category"`
	MaxProgress int       `json:"max_progress" db:"max_progress"` // For progress-based achievements
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UserAchievement struct {
	UserID        int        `json:"user_id" db:"user_id"`
	AchievementID string     `json:"achievement_id" db:"achievement_id"`
	Progress      int        `json:"progress" db:"progress"`
	Completed     bool       `json:"completed" db:"completed"`
	CompletedAt   *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type UserAchievementView struct {
	Achievement
	Progress    int        `json:"progress"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`
}

type GameActivity struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Type      string    `json:"type" db:"type"` // mystery_solved, badge_earned, record_set, etc.
	Title     string    `json:"title" db:"title"`
	Details   string    `json:"details" db:"details"`
	Icon      string    `json:"icon" db:"icon"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
