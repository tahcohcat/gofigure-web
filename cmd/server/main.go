package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	
	"github.com/tahcohcat/gofigure-web/internal/api"
	"github.com/tahcohcat/gofigure-web/internal/websocket"
)

func main() {
	r := mux.NewRouter()

	// API routes
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	gameHandler := api.RegisterRoutes(apiRouter)
	
	// TTS routes (requires game handler for mystery data access)
	api.RegisterTTSRoutes(apiRouter, gameHandler)

	// WebSocket routes
	websocket.RegisterRoutes(r)

	// Static file serving
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))
	
	// Serve the main page
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	// CORS setup for development
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
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