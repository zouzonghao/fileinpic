package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// sessionManager holds the sessions map and a mutex for concurrent access.
type sessionManager struct {
	sessions map[string]time.Time
	mu       sync.RWMutex
}

// Global session manager
var manager = &sessionManager{
	sessions: make(map[string]time.Time),
}

// init starts a background goroutine to clean up expired sessions periodically.
func init() {
	go manager.cleanupSessions()
}

// Store adds a new session token to the manager.
func (m *sessionManager) Store(token string, expiry time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = expiry
}

// Load retrieves a session's expiry time.
func (m *sessionManager) Load(token string) (time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	expiry, exists := m.sessions[token]
	return expiry, exists
}

// Delete removes a session token from the manager.
func (m *sessionManager) Delete(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

// cleanupSessions iterates through the sessions and removes expired ones.
func (m *sessionManager) cleanupSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		for token, expiry := range m.sessions {
			if time.Now().After(expiry) {
				delete(m.sessions, token)
			}
		}
		m.mu.Unlock()
	}
}

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
		expiry := time.Now().Add(24 * time.Hour)
		manager.Store(sessionToken, expiry)

		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken,
			Expires: expiry,
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
		expiry, exists := manager.Load(sessionToken)

		if !exists || expiry.Before(time.Now()) {
			if exists {
				manager.Delete(sessionToken)
			}
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
