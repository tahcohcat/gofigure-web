// internal/auth/auth.go
package auth

import (
	"encoding/json"
	"html/template"
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
	Store = sessions.NewCookieStore([]byte(viper.GetString("auth.session_secret")))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	userService = us
}

// LoginHandler handles both GET (show form) and POST (process login)
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("web/templates/login.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		// Handle both form and JSON requests
		var req models.LoginRequest

		contentType := r.Header.Get("Content-Type")
		if contentType == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		} else {
			r.ParseForm()
			req.Username = r.FormValue("username")
			req.Password = r.FormValue("password")
		}

		user, err := userService.AuthenticateUser(&req)
		if err != nil {
			if contentType == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
				return
			}

			tmpl, _ := template.ParseFiles("web/templates/login.html")
			tmpl.Execute(w, map[string]string{"Error": "Invalid username or password"})
			return
		}

		// Set session
		session, _ := Store.Get(r, "session-name")
		session.Values["authenticated"] = true
		session.Values["user_id"] = user.ID
		session.Values["username"] = user.Username
		session.Save(r, w)

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

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("web/templates/register.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		var req models.CreateUserRequest

		contentType := r.Header.Get("Content-Type")
		if contentType == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		} else {
			r.ParseForm()
			req.Username = r.FormValue("username")
			req.Email = r.FormValue("email")
			req.Password = r.FormValue("password")
			req.DisplayName = r.FormValue("display_name")
		}

		// Basic validation
		if len(req.Password) < 6 {
			respondWithError(w, r, "Password must be at least 6 characters", http.StatusBadRequest)
			return
		}
		if len(req.Username) < 3 {
			respondWithError(w, r, "Username must be at least 3 characters", http.StatusBadRequest)
			return
		}

		user, err := userService.CreateUser(&req)
		if err != nil {
			respondWithError(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		// Auto-login after registration
		session, _ := Store.Get(r, "session-name")
		session.Values["authenticated"] = true
		session.Values["user_id"] = user.ID
		session.Values["username"] = user.Username
		session.Save(r, w)

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
	userID := GetUserIDFromSession(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		user, err := userService.GetUserByID(userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		stats, err := userService.GetUserStats(userID)
		if err != nil {
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

		tmpl, err := template.ParseFiles("web/templates/profile.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, struct {
			User  *models.User
			Stats *models.UserStats
		}{
			User:  user,
			Stats: stats,
		})
		return
	}

	// Handle profile updates
	if r.Method == http.MethodPost {
		r.ParseForm()
		displayName := r.FormValue("display_name")
		email := r.FormValue("email")

		if err := userService.UpdateProfile(userID, displayName, email); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/profile", http.StatusFound)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// AuthMiddleware ensures the user is authenticated
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := Store.Get(r, "session-name")

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			// Check if it's an API request
			if r.Header.Get("Content-Type") == "application/json" || r.Header.Get("Accept") == "application/json" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

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
	if r.Header.Get("Content-Type") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}

	// For HTML requests, redirect back to the form with error
	if statusCode == http.StatusBadRequest && r.URL.Path == "/register" {
		tmpl, _ := template.ParseFiles("web/templates/register.html")
		tmpl.Execute(w, map[string]string{"Error": message})
		return
	}

	http.Error(w, message, statusCode)
}
