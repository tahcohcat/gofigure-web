package database

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sqlx.DB
}

// NewDB creates a new database connection
func NewDB(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		databaseURL = "users.db" // Default SQLite file
	}

	db, err := sqlx.Connect("sqlite3", databaseURL+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	dbWrapper := &DB{DB: db}

	// Initialize database schema
	if err := dbWrapper.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("Database connection established and tables initialized")
	return dbWrapper, nil
}

// createTables creates the necessary database tables
func (db *DB) createTables() error {
	// Users table
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		display_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login_at DATETIME,
		is_active BOOLEAN DEFAULT TRUE
	);`

	// User stats table
	statsTable := `
	CREATE TABLE IF NOT EXISTS user_stats (
		user_id INTEGER PRIMARY KEY,
		games_played INTEGER DEFAULT 0,
		games_won INTEGER DEFAULT 0,
		total_play_time INTEGER DEFAULT 0,
		fastest_solve INTEGER DEFAULT 0,
		favorite_mystery TEXT DEFAULT '',
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	// User game sessions table
	sessionsTable := `
	CREATE TABLE IF NOT EXISTS user_game_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		mystery_id TEXT NOT NULL,
		session_id TEXT UNIQUE NOT NULL,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME,
		solved BOOLEAN,
		time_spent INTEGER,
		questions_asked INTEGER,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	// Create indexes for better performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON user_game_sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON user_game_sessions(session_id);`,
	}

	// Execute table creation
	for _, query := range []string{usersTable, statsTable, sessionsTable} {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes
	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	if err := db.CreateAchievementTables(); err != nil {
		return fmt.Errorf("failed to create achievement tables: %w", err)
	}

	return nil
}

// Database migration for achievement system
func (db *DB) CreateAchievementTables() error {
	// Achievements table
	achievementsTable := `
	CREATE TABLE IF NOT EXISTS achievements (
		id TEXT PRIMARY KEY,
		icon TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		type TEXT NOT NULL, -- milestone, challenge, progress, special, collection, mastery
		category TEXT NOT NULL DEFAULT 'general',
		max_progress INTEGER DEFAULT 0, -- 0 for binary achievements, >0 for progress-based
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// User achievements table
	userAchievementsTable := `
	CREATE TABLE IF NOT EXISTS user_achievements (
		user_id INTEGER NOT NULL,
		achievement_id TEXT NOT NULL,
		progress INTEGER DEFAULT 0,
		completed BOOLEAN DEFAULT FALSE,
		completed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, achievement_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (achievement_id) REFERENCES achievements(id) ON DELETE CASCADE
	);`

	// Game activities table
	activitiesTable := `
	CREATE TABLE IF NOT EXISTS game_activities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		type TEXT NOT NULL, -- mystery_solved, badge_earned, record_set, etc.
		title TEXT NOT NULL,
		details TEXT DEFAULT '',
		icon TEXT DEFAULT '🎯',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_user_achievements_user_id ON user_achievements(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_achievements_completed ON user_achievements(completed, completed_at);`,
		`CREATE INDEX IF NOT EXISTS idx_game_activities_user_id ON game_activities(user_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_achievements_type ON achievements(type);`,
	}

	// Execute table creation
	for _, query := range []string{achievementsTable, userAchievementsTable, activitiesTable} {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create achievement table: %w", err)
		}
	}

	// Create indexes
	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create achievement index: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
