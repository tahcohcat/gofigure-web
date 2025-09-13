package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/spf13/viper"

	"github.com/tahcohcat/gofigure-web/internal/api"
	"github.com/tahcohcat/gofigure-web/internal/auth"
	"github.com/tahcohcat/gofigure-web/internal/credits"
	"github.com/tahcohcat/gofigure-web/internal/database"
	"github.com/tahcohcat/gofigure-web/internal/services"
	"github.com/tahcohcat/gofigure-web/internal/websocket"

	// Import SQLite driver
	_ "github.com/mattn/go-sqlite3"
)

func setupViper() {
	// Read base config file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Error reading config.yaml: %s", err)
		log.Println("Using default configuration")
	}

	// Read local config file for overrides (ignored by git)
	viper.SetConfigName("config.local")
	if err := viper.MergeInConfig(); err != nil {
		log.Printf("No local config found (this is normal): %s", err)
	}

	// Set default values
	viper.SetDefault("auth.session_secret", "change-this-secret-in-production")
	viper.SetDefault("auth.disabled", false)
	viper.SetDefault("auth.login_password", "")
	viper.SetDefault("database.url", "users.db")

	// Read environment variables
	viper.SetEnvPrefix("GOFIGURE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

func main() {
	// Load configuration
	setupViper()

	// Initialize database
	db, err := database.NewDB(viper.GetString("database.url"))
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	userService := services.NewUserService(db)

	// Initialize auth with user service
	auth.Init(userService)

	// Setup router
	r := mux.NewRouter()

	// Public routes (no authentication required)
	r.HandleFunc("/login", auth.LoginHandler).Methods("GET", "POST")
	r.HandleFunc("/register", auth.RegisterHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", auth.LogoutHandler).Methods("GET", "POST")
	r.HandleFunc("/credits", credits.Handler)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Authenticated routes
	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(auth.AuthMiddleware)

	authRouter.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/profile.html")
	}).Methods("GET")

	// API routes with user service integration
	apiRouter := authRouter.PathPrefix("/api/v1").Subrouter()
	gameHandler := api.RegisterRoutes(apiRouter, userService)

	// TTS routes (requires game handler for mystery data access)
	api.RegisterTTSRoutes(apiRouter, gameHandler)

	// WebSocket routes
	websocket.RegisterRoutes(authRouter)

	// Serve the main page
	authRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	// API routes for user management
	apiRouter.HandleFunc("/auth/profile", api.GetUserProfile(userService)).Methods("GET")
	apiRouter.HandleFunc("/auth/profile", api.UpdateUserProfile(userService)).Methods("PUT")
	apiRouter.HandleFunc("/auth/password", api.ChangePassword(userService)).Methods("PUT")

	// CORS setup for development
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🎭 GoFigure Web Server starting on port %s", port)
	log.Printf("📍 Open http://localhost:%s in your browser", port)

	// Log auth configuration
	if viper.GetBool("auth.disabled") {
		log.Printf("⚠️  Authentication is DISABLED for development")
	} else {
		log.Printf("🔐 Authentication is ENABLED")
		log.Printf("📝 Registration: http://localhost:%s/register", port)
		log.Printf("🔑 Login: http://localhost:%s/login", port)
	}

	// Log database info
	log.Printf("💾 Database: %s", viper.GetString("database.url"))

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
