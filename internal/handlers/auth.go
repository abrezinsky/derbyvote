package handlers

import (
	"net/http"

	"github.com/abrezinsky/derbyvote/internal/auth"
)

// LoginPageData holds data for the login template
type LoginPageData struct {
	Error string
}

// handleLoginPage renders the login form
func (h *Handlers) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to admin
	if h.Auth.GetSessionFromRequest(r) {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	h.templates.AdminLogin.Execute(w, LoginPageData{})
}

// handleLogin processes login form submission
func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")

	token, ok := h.Auth.Login(password)
	if !ok {
		h.templates.AdminLogin.Execute(w, LoginPageData{
			Error: "Invalid password",
		})
		return
	}

	auth.SetSessionCookie(w, token)
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// handleLogout clears the session and redirects to login
func (h *Handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get and invalidate the session
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		h.Auth.Logout(cookie.Value)
	}

	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}
