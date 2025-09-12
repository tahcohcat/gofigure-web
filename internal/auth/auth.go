// internal/auth/auth.go - Updated with better error handling and logging
package auth

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	"github.com/tahcohcat/gofigure-web/internal/models"
	"github.com/tahcohcat/gofigure-web/internal/services"
)

var (
	Store       *sessions.CookieStore
	userService *services.UserService
)

func Init(us *services.UserService) {
	log.Printf("🔐 Initializing auth system...")

	sessionSecret := viper.GetString("auth.session_secret")
	if sessionSecret == "" {
		log.Printf("⚠️  WARNING: No session secret configured, using default (INSECURE for production)")
		sessionSecret = "default-insecure-key-change-this"
	}

	Store = sessions.NewCookieStore([]byte(sessionSecret))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	userService = us

	log.Printf("✅ Auth system initialized successfully")
}

// LoginHandler handles both GET (show form) and POST (process login)
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 Login handler called: %s %s", r.Method, r.URL.Path)

	if r.Method == http.MethodGet {
		log.Printf("📄 Serving login page")

		// Try multiple template paths
		templatePaths := []string{
			"web/templates/login.html",
			"web/login.html",
			"./web/templates/login.html",
			"./web/login.html",
		}

		var tmpl *template.Template
		var err error

		for _, path := range templatePaths {
			tmpl, err = template.ParseFiles(path)
			if err == nil {
				log.Printf("✅ Found login template at: %s", path)
				break
			}
			log.Printf("❌ Template not found at: %s - %v", path, err)
		}

		if err != nil {
			log.Printf("❌ Failed to find login template in any location: %v", err)
			// Fallback: serve a basic HTML form
			fallbackHTML := `
<!DOCTYPE html>
<html>
<head><title>Login - GoFigure</title></head>
<body>
	<h1>Login to GoFigure</h1>
	<form method="POST" action="/login">
		<div>
			<label>Username:</label>
			<input type="text" name="username" required>
		</div>
		<div>
			<label>Password:</label>
			<input type="password" name="password" required>
		</div>
		<button type="submit">Login</button>
	</form>
	<p><a href="/register">Don't have an account? Register here</a></p>
</body>
</html>`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fallbackHTML))
			return
		}

		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("❌ Failed to execute login template: %v", err)
			http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Login page served successfully")
		return
	}

	if r.Method == http.MethodPost {
		log.Printf("📝 Processing login form submission")

		if userService == nil {
			log.Printf("❌ User service not initialized!")
			http.Error(w, "User service not available", http.StatusInternalServerError)
			return
		}

		// Handle both form and JSON requests
		var req models.LoginRequest

		contentType := r.Header.Get("Content-Type")
		log.Printf("📋 Content-Type: %s", contentType)

		if contentType == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("❌ Failed to decode JSON request: %v", err)
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		} else {
			if err := r.ParseForm(); err != nil {
				log.Printf("❌ Failed to parse form: %v", err)
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}
			req.Username = r.FormValue("username")
			req.Password = r.FormValue("password")
		}

		log.Printf("👤 Attempting login for username: %s", req.Username)

		user, err := userService.AuthenticateUser(&req)
		if err != nil {
			log.Printf("❌ Authentication failed for %s: %v", req.Username, err)

			if contentType == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
				return
			}

			// For HTML, try to render template with error
			templatePaths := []string{
				"web/templates/login.html",
				"web/login.html",
			}

			var tmpl *template.Template
			for _, path := range templatePaths {
				tmpl, err = template.ParseFiles(path)
				if err == nil {
					break
				}
			}

			if tmpl != nil {
				tmpl.Execute(w, map[string]string{"Error": "Invalid username or password"})
			} else {
				http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			}
			return
		}

		log.Printf("✅ Authentication successful for user: %s (ID: %d)", user.Username, user.ID)

		// Set session
		session, err := Store.Get(r, "session-name")
		if err != nil {
			log.Printf("❌ Failed to get session: %v", err)
			http.Error(w, "Session error", http.StatusInternalServerError)
			return
		}

		session.Values["authenticated"] = true
		session.Values["user_id"] = user.ID
		session.Values["username"] = user.Username

		if err := session.Save(r, w); err != nil {
			log.Printf("❌ Failed to save session: %v", err)
			http.Error(w, "Failed to save session", http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Session created for user: %s", user.Username)

		if contentType == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"user": map[string]interface{}{
					"id":           user.ID,
					"username":     user.Username,
					"display_name": user.DisplayName,
				},
			})
			return
		}

		log.Printf("🔄 Redirecting to home page")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	log.Printf("❌ Method not allowed: %s", r.Method)
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("📝 Register handler called: %s %s", r.Method, r.URL.Path)

	if r.Method == http.MethodGet {
		log.Printf("📄 Serving registration page")

		templatePaths := []string{
			"web/templates/register.html",
			"web/register.html",
		}

		var tmpl *template.Template
		var err error

		for _, path := range templatePaths {
			tmpl, err = template.ParseFiles(path)
			if err == nil {
				log.Printf("✅ Found register template at: %s", path)
				break
			}
		}

		if err != nil {
			log.Printf("❌ Failed to find register template: %v", err)
			// Fallback HTML
			fallbackHTML := `
<!DOCTYPE html>
<html>
<head><title>Register - GoFigure</title></head>
<body>
	<h1>Create Account</h1>
	<form method="POST" action="/register">
		<div>
			<label>Username:</label>
			<input type="text" name="username" required>
		</div>
		<div>
			<label>Display Name:</label>
			<input type="text" name="display_name" required>
		</div>
		<div>
			<label>Email:</label>
			<input type="email" name="email" required>
		</div>
		<div>
			<label>Password:</label>
			<input type="password" name="password" required>
		</div>
		<button type="submit">Create Account</button>
	</form>
	<p><a href="/login">Already have an account? Login here</a></p>
</body>
</html>`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fallbackHTML))
			return
		}

		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("❌ Failed to execute register template: %v", err)
			http.Error(w, "Template execution error", http.StatusInternalServerError)
			return
		}

		return
	}

	if r.Method == http.MethodPost {
		log.Printf("📝 Processing registration form submission")

		if userService == nil {
			log.Printf("❌ User service not initialized!")
			http.Error(w, "User service not available", http.StatusInternalServerError)
			return
		}

		var req models.CreateUserRequest

		contentType := r.Header.Get("Content-Type")
		if contentType == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("❌ Failed to decode JSON request: %v", err)
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		} else {
			if err := r.ParseForm(); err != nil {
				log.Printf("❌ Failed to parse form: %v", err)
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}
			req.Username = r.FormValue("username")
			req.Email = r.FormValue("email")
			req.Password = r.FormValue("password")
			req.DisplayName = r.FormValue("display_name")
		}

		log.Printf("👤 Attempting registration for username: %s, email: %s", req.Username, req.Email)

		// Basic validation
		if len(req.Password) < 6 {
			log.Printf("❌ Password too short for user: %s", req.Username)
			respondWithError(w, r, "Password must be at least 6 characters", http.StatusBadRequest)
			return
		}
		if len(req.Username) < 3 {
			log.Printf("❌ Username too short: %s", req.Username)
			respondWithError(w, r, "Username must be at least 3 characters", http.StatusBadRequest)
			return
		}

		user, err := userService.CreateUser(&req)
		if err != nil {
			log.Printf("❌ User creation failed: %v", err)
			respondWithError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("✅ User created successfully: %s (ID: %d)", user.Username, user.ID)

		// Auto-login after registration
		session, err := Store.Get(r, "session-name")
		if err != nil {
			log.Printf("❌ Failed to get session after registration: %v", err)
		} else {
			session.Values["authenticated"] = true
			session.Values["user_id"] = user.ID
			session.Values["username"] = user.Username
			session.Save(r, w)
			log.Printf("✅ Auto-login session created for new user: %s", user.Username)
		}

		if contentType == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"user": map[string]interface{}{
					"id":           user.ID,
					"username":     user.Username,
					"display_name": user.DisplayName,
				},
			})
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// LogoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("🚪 Logout handler called")

	session, _ := Store.Get(r, "session-name")
	session.Values["authenticated"] = false
	session.Values["user_id"] = nil
	session.Values["username"] = nil
	session.Save(r, w)

	// Check if it's an API request
	if r.Header.Get("Content-Type") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// ProfileHandler shows user profile
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("👤 Profile handler called: %s", r.Method)

	userID := GetUserIDFromSession(r)
	if userID == 0 {
		log.Printf("❌ No user ID in session")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		user, err := userService.GetUserByID(userID)
		if err != nil {
			log.Printf("❌ Failed to get user by ID %d: %v", userID, err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		stats, err := userService.GetUserStats(userID)
		if err != nil {
			log.Printf("⚠️  Failed to get user stats, using defaults: %v", err)
			stats = &models.UserStats{UserID: userID} // Default empty stats
		}

		if r.Header.Get("Content-Type") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user":  user,
				"stats": stats,
			})
			return
		}

		// Try to load profile template
		templatePaths := []string{
			"web/templates/profile.html",
			"web/profile.html",
		}

		var tmpl *template.Template
		for _, path := range templatePaths {
			tmpl, err = template.ParseFiles(path)
			if err == nil {
				break
			}
		}

		if err != nil {
			log.Printf("❌ Failed to find profile template: %v", err)
			// Return JSON as fallback
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user":  user,
				"stats": stats,
			})
			return
		}

		if err := tmpl.Execute(w, struct {
			User  *models.User
			Stats *models.UserStats
		}{
			User:  user,
			Stats: stats,
		}); err != nil {
			log.Printf("❌ Failed to execute profile template: %v", err)
			http.Error(w, "Template execution error", http.StatusInternalServerError)
			return
		}

		return
	}

	// Handle profile updates
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		displayName := r.FormValue("display_name")
		email := r.FormValue("email")

		if err := userService.UpdateProfile(userID, displayName, email); err != nil {
			log.Printf("❌ Failed to update profile for user %d: %v", userID, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("✅ Profile updated for user %d", userID)
		http.Redirect(w, r, "/profile", http.StatusFound)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// AuthMiddleware ensures the user is authenticated
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔒 Auth middleware checking: %s %s", r.Method, r.URL.Path)

		session, err := Store.Get(r, "session-name")
		if err != nil {
			log.Printf("❌ Failed to get session: %v", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		auth, ok := session.Values["authenticated"].(bool)
		if !ok {
			log.Printf("❌ No authentication value in session")
		}

		userID, userOK := session.Values["user_id"].(int)
		if !userOK {
			log.Printf("❌ No user_id in session")
		}

		if !ok || !auth || userID == 0 {
			log.Printf("❌ Authentication failed: auth=%v, userID=%d", auth, userID)
			// Check if it's an API request
			if r.Header.Get("Content-Type") == "application/json" || r.Header.Get("Accept") == "application/json" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		log.Printf("✅ Authentication successful for user %d", userID)
		next.ServeHTTP(w, r)
	})
}

// GetUserFromSession retrieves the current user from the session
func GetUserFromSession(r *http.Request) *models.User {
	session, _ := Store.Get(r, "session-name")

	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return nil
	}

	user, err := userService.GetUserByID(userID)
	if err != nil {
		log.Printf("❌ Failed to get user from session: %v", err)
		return nil
	}

	return user
}

// GetUserIDFromSession retrieves the current user ID from the session
func GetUserIDFromSession(r *http.Request) int {
	session, _ := Store.Get(r, "session-name")

	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return 0
	}

	return userID
}

// Helper function to respond with errors in both JSON and HTML formats
func respondWithError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	log.Printf("❌ Responding with error: %s (status: %d)", message, statusCode)

	if r.Header.Get("Content-Type") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}

	// For HTML requests, redirect back to the form with error
	if statusCode == http.StatusBadRequest && r.URL.Path == "/register" {
		templatePaths := []string{
			"web/templates/register.html",
			"web/register.html",
		}

		var tmpl *template.Template
		for _, path := range templatePaths {
			tmpl, _ = template.ParseFiles(path)
			if tmpl != nil {
				break
			}
		}

		if tmpl != nil {
			tmpl.Execute(w, map[string]string{"Error": message})
			return
		}
	}

	http.Error(w, message, statusCode)
}
