// cmd/server/main.go - Updated with better error handling and logging
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
	log.Printf("📋 Setting up configuration...")

	// Read base config file
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("⚠️  No config file found, using defaults and environment variables")
		} else {
			log.Printf("⚠️  Error reading config file: %s", err)
		}
	} else {
		log.Printf("✅ Config file loaded: %s", viper.ConfigFileUsed())
	}

	// Read local config file for overrides (ignored by git)
	viper.SetConfigName("config.local")
	if err := viper.MergeInConfig(); err != nil {
		log.Printf("ℹ️  No local config file found (this is normal)")
	} else {
		log.Printf("✅ Local config file merged")
	}

	// Read environment variables
	viper.SetEnvPrefix("GOFIGURE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("auth.session_secret", "your-secret-key-change-this-in-production")
	viper.SetDefault("database.path", "./gofigure.db")

	log.Printf("✅ Configuration setup complete")
}

func main() {
	log.Printf("🎭 Starting GoFigure Web Server...")

	// Load config from files and environment variables
	setupViper()

	// Initialize database
	dbPath := viper.GetString("database.path")
	log.Printf("🗄️  Initializing database at: %s", dbPath)

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("✅ Database initialized successfully")

	// Initialize services
	log.Printf("⚙️  Initializing user service...")
	userService := services.NewUserService(db)

	// Test database connection by checking if we can access users table
	if _, err := userService.GetUserByID(1); err != nil {
		log.Printf("ℹ️  Database appears empty or user #1 not found (this is normal for new installations)")
	}

	log.Printf("✅ User service initialized successfully")

	// Initialize auth with user service
	log.Printf("🔐 Initializing authentication...")
	auth.Init(userService)

	// Setup router
	log.Printf("🛣️  Setting up routes...")
	r := mux.NewRouter()

	// Add logging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("📡 %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			next.ServeHTTP(w, r)
		})
	})

	// Public routes (no authentication required)
	log.Printf("🌐 Setting up public routes...")
	publicRouter := r.PathPrefix("/").Subrouter()
	publicRouter.HandleFunc("/login", auth.LoginHandler).Methods("GET", "POST")
	publicRouter.HandleFunc("/register", auth.RegisterHandler).Methods("GET", "POST")
	publicRouter.HandleFunc("/logout", auth.LogoutHandler).Methods("POST", "GET")
	publicRouter.HandleFunc("/credits", credits.Handler).Methods("GET")
	publicRouter.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Test route to check if server is working
	publicRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "service": "gofigure-web"}`))
	}).Methods("GET")

	// Authenticated routes
	log.Printf("🔒 Setting up authenticated routes...")
	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(auth.AuthMiddleware)

	// Profile route
	authRouter.HandleFunc("/profile", auth.ProfileHandler).Methods("GET", "POST")

	// API routes
	log.Printf("🔌 Setting up API routes...")
	apiRouter := authRouter.PathPrefix("/api/v1").Subrouter()
	gameHandler := api.RegisterRoutes(apiRouter, userService) // Pass userService to API

	// TTS routes (requires game handler for mystery data access)
	api.RegisterTTSRoutes(apiRouter, gameHandler)

	// WebSocket routes
	log.Printf("📡 Setting up WebSocket routes...")
	websocket.RegisterRoutes(authRouter)

	// Serve the main page
	authRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🏠 Serving main page to authenticated user")
		http.ServeFile(w, r, "./web/templates/index.html")
	}).Methods("GET")

	// Redirect root to login if not authenticated
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🏠 Redirecting unauthenticated user to login")
		http.Redirect(w, r, "/login", http.StatusFound)
	}).Methods("GET")

	// CORS setup for development
	log.Printf("🌍 Setting up CORS...")
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
	log.Printf("🔐 Session secret configured: %v", viper.GetString("auth.session_secret") != "")

	// Check if templates exist
	templatePaths := []string{
		"./web/templates/login.html",
		"./web/templates/register.html",
		"./web/templates/index.html",
		"./web/login.html",
		"./web/register.html",
	}

	log.Printf("📄 Checking template files...")
	for _, path := range templatePaths {
		if _, err := os.Stat(path); err == nil {
			log.Printf("✅ Found template: %s", path)
		} else {
			log.Printf("❌ Missing template: %s", path)
		}
	}

	// Check static files
	if _, err := os.Stat("./web/static"); err == nil {
		log.Printf("✅ Static files directory exists")
	} else {
		log.Printf("❌ Missing static files directory: ./web/static")
	}

	log.Printf("🚀 Server ready! Starting HTTP server...")
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}
