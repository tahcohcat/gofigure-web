package auth

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

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
	// Initialize session store
	sessionSecret := viper.GetString("auth.session_secret")
	if sessionSecret == "" {
		sessionSecret = "default-secret-key-change-in-production"
	}
	Store = sessions.NewCookieStore([]byte(sessionSecret))

	// Set user service
	userService = us
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("web/login.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Check if using legacy admin password (fallback)
		configPassword := viper.GetString("auth.login_password")
		if configPassword != "" && email == "" && password == configPassword {
			session, _ := Store.Get(r, "session-name")
			session.Values["authenticated"] = true
			session.Values["username"] = "admin"
			session.Values["user_id"] = 0 // Special admin user ID
			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// User authentication
		if email != "" && password != "" {
			loginReq := &models.LoginRequest{
				Email: email,
				Password: password,
			}

			user, err := userService.AuthenticateUser(loginReq)
			if err == nil && user != nil {
				session, _ := Store.Get(r, "session-name")
				session.Values["authenticated"] = true
				session.Values["username"] = user.Username
				session.Values["user_id"] = user.ID
				session.Save(r, w)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}

		// Authentication failed
		tmpl, _ := template.ParseFiles("web/login.html")
		tmpl.Execute(w, map[string]string{"Error": "Invalid credentials"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("web/register.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		// Check if it's JSON or form data
		contentType := r.Header.Get("Content-Type")

		var req models.CreateUserRequest
		var err error

		if contentType == "application/json" {
			// Handle JSON request (for API)
			err = json.NewDecoder(r.Body).Decode(&req)
		} else {
			// Handle form data (for web)
			r.ParseForm()
			req = models.CreateUserRequest{
				Username: strings.TrimSpace(r.FormValue("username")),

				Email:       strings.TrimSpace(r.FormValue("email")),
				Password:    r.FormValue("password"),
				DisplayName: strings.TrimSpace(r.FormValue("display_name")),
			}

			// Use username as display name if not provided
			if req.DisplayName == "" {
				req.DisplayName = req.Username
			}

			// Debug logging to help identify the issue
			log.Printf("Registration form data - Username: '%s', Email: '%s', Password length: %d, DisplayName: '%s'",
				req.Username, req.Email, len(req.Password), req.DisplayName)

			// Basic form validation
			confirmPassword := r.FormValue("confirm_password")
			if req.Password != confirmPassword {
				tmpl, _ := template.ParseFiles("web/register.html")
				tmpl.Execute(w, map[string]string{
					"Error":       "Passwords do not match",
					"Username":    req.Username,
					"Email":       req.Email,
					"DisplayName": req.DisplayName,
				})
				return
			}
		}

		if err != nil {
			if contentType == "application/json" {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
			} else {
				tmpl, _ := template.ParseFiles("web/register.html")
				tmpl.Execute(w, map[string]string{"Error": "Invalid form data"})
			}
			return
		}

		// Basic validation
		if req.Username == "" || req.Password == "" || req.Email == "" {
			errorMsg := "All fields are required"
			if contentType == "application/json" {
				http.Error(w, errorMsg, http.StatusBadRequest)
			} else {
				tmpl, _ := template.ParseFiles("web/register.html")
				tmpl.Execute(w, map[string]string{
					"Error":    errorMsg,
					"Username": req.Username,
					"Email":    req.Email,
				})
			}
			return
		}

		if len(req.Password) < 6 {
			errorMsg := "Password must be at least 6 characters"
			if contentType == "application/json" {
				http.Error(w, errorMsg, http.StatusBadRequest)
			} else {
				tmpl, _ := template.ParseFiles("web/register.html")
				tmpl.Execute(w, map[string]string{
					"Error":    errorMsg,
					"Username": req.Username,
					"Email":    req.Email,
				})
			}
			return
		}

		// Create user
		user, err := userService.CreateUser(&req)
		if err != nil {
			errorMsg := err.Error()
			if contentType == "application/json" {
				http.Error(w, errorMsg, http.StatusBadRequest)
			} else {
				tmpl, _ := template.ParseFiles("web/register.html")
				tmpl.Execute(w, map[string]string{
					"Error":    errorMsg,
					"Username": req.Username,
					"Email":    req.Email,
				})
			}
			return
		}

		// Handle successful registration
		if contentType == "application/json" {
			// API response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"user": map[string]interface{}{
					"id":           user.ID,
					"username":     user.Username,
					"email":        user.Email,
					"display_name": user.DisplayName,
				},
			})
		} else {
			// Web form - auto-login and redirect
			session, _ := Store.Get(r, "session-name")
			session.Values["authenticated"] = true
			session.Values["username"] = user.Username
			session.Values["user_id"] = user.ID
			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := Store.Get(r, "session-name")
	session.Values["authenticated"] = false
	session.Values["username"] = nil
	session.Values["user_id"] = nil
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if auth is disabled in config (for development)
		if viper.GetBool("auth.disabled") {
			next.ServeHTTP(w, r)
			return
		}

		session, _ := Store.Get(r, "session-name")

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserIDFromSession extracts the user ID from the session
func GetUserIDFromSession(r *http.Request) int {
	session, err := Store.Get(r, "session-name")
	if err != nil {
		return 0
	}

	if userID, ok := session.Values["user_id"].(int); ok {
		return userID
	}

	return 0
}

// GetUsernameFromSession extracts the username from the session
func GetUsernameFromSession(r *http.Request) string {
	session, err := Store.Get(r, "session-name")
	if err != nil {
		return ""
	}

	if username, ok := session.Values["username"].(string); ok {
		return username
	}

	return ""
}
