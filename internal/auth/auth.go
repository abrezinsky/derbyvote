package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	CookieName     = "derbyvote_session"
	SessionExpiry  = 24 * time.Hour
)

// Derby-themed words for password generation
var derbyWords = []string{
	"pinewood", "racer", "derby", "scout", "trophy",
	"wheels", "axle", "champion", "finish", "speed",
	"pack", "cub", "arrow", "blue", "gold",
	"tiger", "wolf", "bear", "webelos",
}

// Auth handles admin authentication
type Auth struct {
	password string
	sessions map[string]time.Time
	mu       sync.RWMutex
}

// New creates a new Auth instance with the given password
func New(password string) *Auth {
	return &Auth{
		password: password,
		sessions: make(map[string]time.Time),
	}
}

// GeneratePassword creates a random 3-word password
func GeneratePassword() string {
	words := make([]string, 3)
	for i := range words {
		idx := randomInt(len(derbyWords))
		words[i] = derbyWords[idx]
	}
	return strings.Join(words, "-")
}

// Login validates the password and returns a session token if valid
func (a *Auth) Login(password string) (string, bool) {
	if password != a.password {
		return "", false
	}

	token := generateToken()
	a.mu.Lock()
	a.sessions[token] = time.Now().Add(SessionExpiry)
	a.mu.Unlock()

	return token, true
}

// Logout invalidates a session token
func (a *Auth) Logout(token string) {
	a.mu.Lock()
	delete(a.sessions, token)
	a.mu.Unlock()
}

// ValidateSession checks if a session token is valid
func (a *Auth) ValidateSession(token string) bool {
	a.mu.RLock()
	expiry, exists := a.sessions[token]
	a.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		a.mu.Lock()
		delete(a.sessions, token)
		a.mu.Unlock()
		return false
	}

	return true
}

// GetSessionFromRequest extracts and validates the session from a request
func (a *Auth) GetSessionFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return false
	}
	return a.ValidateSession(cookie.Value)
}

// RequireAuth middleware for admin pages (redirects to login)
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.GetSessionFromRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, "/admin/login", http.StatusFound)
	})
}

// RequireAuthAPI middleware for API endpoints (returns 401)
func (a *Auth) RequireAuthAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.GetSessionFromRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":"UNAUTHORIZED","error":"Unauthorized - please log in"}`))
	})
}

// SetSessionCookie sets the session cookie on the response
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(SessionExpiry.Seconds()),
	})
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// generateToken creates a random session token
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// randomInt returns a random int in [0, max)
func randomInt(max int) int {
	bytes := make([]byte, 1)
	rand.Read(bytes)
	return int(bytes[0]) % max
}
