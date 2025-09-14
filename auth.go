package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var sessions = map[string]time.Time{}

func loginHandler(config AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if creds.Password != config.Password {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		sessionToken := uuid.NewString()
		sessions[sessionToken] = time.Now().Add(24 * time.Hour)

		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken,
			Expires: sessions[sessionToken],
			Path:    "/",
		})
	}
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err != nil {
			if err == http.ErrNoCookie {
				http.Redirect(w, r, "/login.html", http.StatusFound)
				return
			}
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		sessionToken := c.Value
		expiry, exists := sessions[sessionToken]

		if !exists || expiry.Before(time.Now()) {
			delete(sessions, sessionToken)
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
