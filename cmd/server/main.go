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
}

func main() {
	// Load config from files and environment variables
	setupViper()

	// Initialize auth
	auth.Init()

	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/login", auth.LoginHandler)
	r.HandleFunc("/logout", auth.LogoutHandler)
	r.HandleFunc("/credits", credits.Handler)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	// Authenticated routes
	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(auth.AuthMiddleware)

	// API routes
	apiRouter := authRouter.PathPrefix("/api/v1").Subrouter()
	gameHandler := api.RegisterRoutes(apiRouter)

	// TTS routes (requires game handler for mystery data access)
	api.RegisterTTSRoutes(apiRouter, gameHandler)

	// WebSocket routes
	websocket.RegisterRoutes(authRouter)

	// Serve the main page
	authRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

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

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
