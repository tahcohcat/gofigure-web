package auth

import (
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
)

var Store *sessions.CookieStore

func Init() {
	Store = sessions.NewCookieStore([]byte(viper.GetString("auth.session_secret")))
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
		password := r.FormValue("password")

		if password == viper.GetString("auth.login_password") {
			session, _ := Store.Get(r, "session-name")
			session.Values["authenticated"] = true
			session.Save(r, w)
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			tmpl, _ := template.ParseFiles("web/login.html")
			tmpl.Execute(w, map[string]string{"Error": "Invalid password"})
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := Store.Get(r, "session-name")
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//todo: always disable locally
		//session, _ := Store.Get(r, "session-name")
		//
		//if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		//	http.Redirect(w, r, "/login", http.StatusSeeOther)
		//	return
		//}

		next.ServeHTTP(w, r)
	})
}
