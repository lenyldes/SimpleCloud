package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

func TestAuthHandlers_Authentication(t *testing.T) {
	svc := newStubAuthService()

	testUser := auth.User{
		ID:         uuid.New(),
		Email:      "tester@example.com",
		Role:       "user",
		QuotaBytes: 1 << 30,
	}
	const testPassword = "test-password-123"
	svc.AddUser(testUser, testPassword)

	handler := auth.NewAuthHandler(svc)

	t.Run("Login success sets HttpOnly cookie", func(t *testing.T) {
		loginPayload := map[string]string{
			"email":    testUser.Email,
			"password": testPassword,
		}
		body, _ := json.Marshal(loginPayload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.LoginHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d, body: %s", rec.Code, rec.Body.String())
		}

		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == auth.SessionCookieName {
				sessionCookie = c
				break
			}
		}

		if sessionCookie == nil {
			t.Fatal("expected simplecloud_session cookie to be set")
		}

		if !sessionCookie.HttpOnly {
			t.Error("expected session cookie to be HttpOnly")
		}
	})

	t.Run("Login failure returns 401", func(t *testing.T) {
		loginPayload := map[string]string{
			"email":    testUser.Email,
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(loginPayload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.LoginHandler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Login invalid body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()

		handler.LoginHandler(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	t.Run("Login method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
		rec := httptest.NewRecorder()

		handler.LoginHandler(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", rec.Code)
		}
	})

	t.Run("Login body > 1MB returns 400 or 413", func(t *testing.T) {
		hugePayload := strings.Repeat("A", 1024*1024+100)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(hugePayload))
		rec := httptest.NewRecorder()
		handler.LoginHandler(rec, req)
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected 400 or 413 for login body > 1MB, got %d", rec.Code)
		}
	})

	t.Run("Login cookie has Secure flag by default or when COOKIE_SECURE is true", func(t *testing.T) {
		t.Setenv("COOKIE_SECURE", "true")
		loginPayload := map[string]string{
			"email":    testUser.Email,
			"password": testPassword,
		}
		body, _ := json.Marshal(loginPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.LoginHandler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var cookie *http.Cookie
		for _, c := range rec.Result().Cookies() {
			if c.Name == auth.SessionCookieName {
				cookie = c
				break
			}
		}
		if cookie == nil || !cookie.Secure {
			t.Errorf("expected cookie to have Secure=true when COOKIE_SECURE is true, got cookie: %+v", cookie)
		}
	})

	t.Run("Login cookie does not have Secure flag when COOKIE_SECURE is false", func(t *testing.T) {
		t.Setenv("COOKIE_SECURE", "false")
		loginPayload := map[string]string{
			"email":    testUser.Email,
			"password": testPassword,
		}
		body, _ := json.Marshal(loginPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.LoginHandler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var cookie *http.Cookie
		for _, c := range rec.Result().Cookies() {
			if c.Name == auth.SessionCookieName {
				cookie = c
				break
			}
		}
		if cookie != nil && cookie.Secure {
			t.Errorf("expected cookie to have Secure=false when COOKIE_SECURE is false, got cookie: %+v", cookie)
		}
	})

	t.Run("AuthHandler cookie Expires matches configured TTL", func(t *testing.T) {
		customTTL := 1 * time.Hour
		h := auth.NewAuthHandlerWithTTL(svc, customTTL)
		loginPayload := map[string]string{
			"email":    testUser.Email,
			"password": testPassword,
		}
		body, _ := json.Marshal(loginPayload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		h.LoginHandler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var cookie *http.Cookie
		for _, c := range rec.Result().Cookies() {
			if c.Name == auth.SessionCookieName {
				cookie = c
				break
			}
		}
		if cookie == nil {
			t.Fatal("expected session cookie")
		}

		expectedExpiry := time.Now().Add(customTTL)
		diff := cookie.Expires.Sub(expectedExpiry)
		if diff < -10*time.Second || diff > 10*time.Second {
			t.Errorf("expected cookie Expires around %v, got %v (diff: %v)", expectedExpiry, cookie.Expires, diff)
		}
	})
}
