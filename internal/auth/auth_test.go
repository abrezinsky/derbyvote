package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	a := New("test-password")

	if a == nil {
		t.Fatal("expected auth to be created")
	}
	if a.password != "test-password" {
		t.Error("expected password to be set")
	}
	if a.sessions == nil {
		t.Error("expected sessions map to be initialized")
	}
}

func TestGeneratePassword_Format(t *testing.T) {
	pw := GeneratePassword()

	parts := strings.Split(pw, "-")
	if len(parts) != 3 {
		t.Errorf("expected 3 words separated by dashes, got %d parts: %s", len(parts), pw)
	}

	// Verify each part is from derbyWords
	for _, part := range parts {
		found := false
		for _, word := range derbyWords {
			if part == word {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("word %q not in derbyWords list", part)
		}
	}
}

func TestGeneratePassword_Randomness(t *testing.T) {
	// Generate multiple passwords and verify they're not all the same
	passwords := make(map[string]bool)
	for i := 0; i < 10; i++ {
		passwords[GeneratePassword()] = true
	}

	// With 19 words and 3 positions, probability of collision is low
	// Should have at least a few unique passwords
	if len(passwords) < 3 {
		t.Errorf("expected more password variety, got only %d unique passwords", len(passwords))
	}
}

func TestLogin_ValidPassword(t *testing.T) {
	a := New("correct-password")

	token, ok := a.Login("correct-password")

	if !ok {
		t.Error("expected login to succeed with correct password")
	}
	if token == "" {
		t.Error("expected token to be returned")
	}
	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64-char token, got %d chars", len(token))
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	a := New("correct-password")

	token, ok := a.Login("wrong-password")

	if ok {
		t.Error("expected login to fail with wrong password")
	}
	if token != "" {
		t.Error("expected empty token on failed login")
	}
}

func TestLogin_CreatesSession(t *testing.T) {
	a := New("password")

	token, _ := a.Login("password")

	if !a.ValidateSession(token) {
		t.Error("expected session to be valid after login")
	}
}

func TestLogout_InvalidatesSession(t *testing.T) {
	a := New("password")
	token, _ := a.Login("password")

	a.Logout(token)

	if a.ValidateSession(token) {
		t.Error("expected session to be invalid after logout")
	}
}

func TestValidateSession_InvalidToken(t *testing.T) {
	a := New("password")

	if a.ValidateSession("nonexistent-token") {
		t.Error("expected false for nonexistent token")
	}
}

func TestValidateSession_ExpiredSession(t *testing.T) {
	a := New("password")
	token, _ := a.Login("password")

	// Manually expire the session
	a.mu.Lock()
	a.sessions[token] = time.Now().Add(-1 * time.Hour)
	a.mu.Unlock()

	if a.ValidateSession(token) {
		t.Error("expected expired session to be invalid")
	}

	// Verify session was cleaned up
	a.mu.RLock()
	_, exists := a.sessions[token]
	a.mu.RUnlock()
	if exists {
		t.Error("expected expired session to be removed")
	}
}

func TestGetSessionFromRequest_ValidCookie(t *testing.T) {
	a := New("password")
	token, _ := a.Login("password")

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})

	if !a.GetSessionFromRequest(req) {
		t.Error("expected valid session from request")
	}
}

func TestGetSessionFromRequest_NoCookie(t *testing.T) {
	a := New("password")

	req := httptest.NewRequest("GET", "/admin", nil)

	if a.GetSessionFromRequest(req) {
		t.Error("expected false when no cookie present")
	}
}

func TestGetSessionFromRequest_InvalidCookie(t *testing.T) {
	a := New("password")

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "invalid-token"})

	if a.GetSessionFromRequest(req) {
		t.Error("expected false for invalid token")
	}
}

func TestRequireAuth_AllowsValidSession(t *testing.T) {
	a := New("password")
	token, _ := a.Login("password")

	handler := a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireAuth_RedirectsWithoutSession(t *testing.T) {
	a := New("password")

	handler := a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %s", rr.Header().Get("Location"))
	}
}

func TestRequireAuthAPI_AllowsValidSession(t *testing.T) {
	a := New("password")
	token, _ := a.Login("password")

	handler := a.RequireAuthAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/admin/settings", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireAuthAPI_Returns401WithoutSession(t *testing.T) {
	a := New("password")

	handler := a.RequireAuthAPI(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/admin/settings", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("expected JSON content type")
	}
	body := rr.Body.String()
	if !strings.Contains(strings.ToLower(body), "unauthorized") {
		t.Errorf("expected unauthorized error in body, got: %s", body)
	}
	if !strings.Contains(body, "UNAUTHORIZED") {
		t.Errorf("expected UNAUTHORIZED code in body, got: %s", body)
	}
}

func TestSetSessionCookie(t *testing.T) {
	rr := httptest.NewRecorder()

	SetSessionCookie(rr, "test-token")

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != CookieName {
		t.Errorf("expected cookie name %s, got %s", CookieName, cookie.Name)
	}
	if cookie.Value != "test-token" {
		t.Errorf("expected cookie value 'test-token', got %s", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}
	if cookie.Path != "/" {
		t.Errorf("expected path '/', got %s", cookie.Path)
	}
}

func TestClearSessionCookie(t *testing.T) {
	rr := httptest.NewRecorder()

	ClearSessionCookie(rr)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != CookieName {
		t.Errorf("expected cookie name %s, got %s", CookieName, cookie.Name)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1 (delete), got %d", cookie.MaxAge)
	}
}

func TestConcurrentSessionAccess(t *testing.T) {
	a := New("password")

	// Test concurrent logins
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			token, _ := a.Login("password")
			a.ValidateSession(token)
			a.Logout(token)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
