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

func NewDatabase(dbPath string) (*DB, error) {
	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &DB{DB: db}

	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return database, nil
}

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
	userStatsTable := `
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
	gameSessionsTable := `
	CREATE TABLE IF NOT EXISTS user_game_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		mystery_id TEXT NOT NULL,
		session_id TEXT NOT NULL,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME,
		solved BOOLEAN DEFAULT FALSE,
		time_spent INTEGER DEFAULT 0,
		questions_asked INTEGER DEFAULT 0,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);",
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON user_game_sessions(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON user_game_sessions(session_id);",
	}

	tables := []string{usersTable, userStatsTable, gameSessionsTable}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	// Create trigger to update user stats when a game session is completed
	trigger := `
	CREATE TRIGGER IF NOT EXISTS update_user_stats_on_game_complete
	AFTER UPDATE ON user_game_sessions
	WHEN NEW.finished_at IS NOT NULL AND OLD.finished_at IS NULL
	BEGIN
		INSERT OR REPLACE INTO user_stats (
			user_id,
			games_played,
			games_won,
			total_play_time,
			fastest_solve
		)
		SELECT 
			NEW.user_id,
			COALESCE((SELECT games_played FROM user_stats WHERE user_id = NEW.user_id), 0) + 1,
			COALESCE((SELECT games_won FROM user_stats WHERE user_id = NEW.user_id), 0) + 
				CASE WHEN NEW.solved THEN 1 ELSE 0 END,
			COALESCE((SELECT total_play_time FROM user_stats WHERE user_id = NEW.user_id), 0) + NEW.time_spent,
			CASE 
				WHEN NEW.solved AND (
					SELECT fastest_solve FROM user_stats WHERE user_id = NEW.user_id
				) > NEW.time_spent OR (
					SELECT fastest_solve FROM user_stats WHERE user_id = NEW.user_id
				) IS NULL OR (
					SELECT fastest_solve FROM user_stats WHERE user_id = NEW.user_id
				) = 0
				THEN NEW.time_spent
				ELSE COALESCE((SELECT fastest_solve FROM user_stats WHERE user_id = NEW.user_id), 0)
			END;
	END;`

	if _, err := db.Exec(trigger); err != nil {
		log.Printf("Warning: failed to create trigger: %v", err)
	}

	return nil
}
