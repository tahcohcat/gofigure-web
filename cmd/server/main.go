// cmd/server/main.go
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
)

func setupViper() {
	// Read base config file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config.yaml: %s", err)
	}

	// Read local config file for overrides (ignored by git)
	viper.SetConfigName("config.local")
	viper.MergeInConfig() // Merge local config on top of base

	// Read environment variables
	viper.SetEnvPrefix("GOFIGURE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values for auth
	viper.SetDefault("auth.session_secret", "your-secret-key-change-this-in-production")
	viper.SetDefault("database.path", "./gofigure.db")
}

func main() {
	// Load config from files and environment variables
	setupViper()

	// Initialize database
	dbPath := viper.GetString("database.path")
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	userService := services.NewUserService(db)

	// Initialize auth with user service
	auth.Init(userService)

	r := mux.NewRouter()

	// Public routes (no authentication required)
	publicRouter := r.PathPrefix("/").Subrouter()
	publicRouter.HandleFunc("/login", auth.LoginHandler).Methods("GET", "POST")
	publicRouter.HandleFunc("/register", auth.RegisterHandler).Methods("GET", "POST")
	publicRouter.HandleFunc("/logout", auth.LogoutHandler).Methods("POST", "GET")
	publicRouter.HandleFunc("/credits", credits.Handler).Methods("GET")
	publicRouter.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Authenticated routes
	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(auth.AuthMiddleware)

	// Profile route
	authRouter.HandleFunc("/profile", auth.ProfileHandler).Methods("GET", "POST")

	// API routes
	apiRouter := authRouter.PathPrefix("/api/v1").Subrouter()
	gameHandler := api.RegisterRoutes(apiRouter, userService) // Pass userService to API

	// TTS routes (requires game handler for mystery data access)
	api.RegisterTTSRoutes(apiRouter, gameHandler)

	// WebSocket routes
	websocket.RegisterRoutes(authRouter)

	// Serve the main page
	authRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	}).Methods("GET")

	// Redirect root to login if not authenticated
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	}).Methods("GET")

	// CORS setup for development
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🎭 GoFigure Web Server starting on port %s", port)
	log.Printf("📍 Open http://localhost:%s in your browser", port)
	log.Printf("🗄️ Database: %s", dbPath)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
