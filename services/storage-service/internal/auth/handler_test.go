package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

func TestAuthHandlersAndMiddleware(t *testing.T) {
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

	t.Run("RequireAuth Middleware blocks unauthenticated requests", func(t *testing.T) {
		protectedHandler := auth.RequireAuth(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for unauthenticated request, got %d", rec.Code)
		}
	})

	t.Run("RequireAuth Middleware blocks invalid token", func(t *testing.T) {
		protectedHandler := auth.RequireAuth(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for invalid token, got %d", rec.Code)
		}
	})

	t.Run("RequireAuth Middleware allows Bearer token request", func(t *testing.T) {
		validToken := svc.IssueSession(testUser.ID, 24*time.Hour)

		protectedHandler := auth.RequireAuth(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.GetUserIDFromContext(r.Context())
			if !ok || userID == uuid.Nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for valid Bearer token, got %d", rec.Code)
		}
	})

	t.Run("RequireAuth Middleware allows Cookie auth request", func(t *testing.T) {
		validToken := svc.IssueSession(testUser.ID, 24*time.Hour)

		protectedHandler := auth.RequireAuth(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: validToken})
		rec := httptest.NewRecorder()

		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for cookie auth, got %d", rec.Code)
		}
	})

	t.Run("Logout clears session cookie", func(t *testing.T) {
		validToken := svc.IssueSession(testUser.ID, 24*time.Hour)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: validToken})
		rec := httptest.NewRecorder()

		handler.LogoutHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on logout, got %d", rec.Code)
		}

		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == auth.SessionCookieName {
				sessionCookie = c
				break
			}
		}

		if sessionCookie == nil || sessionCookie.MaxAge > 0 {
			t.Error("expected session cookie to be cleared on logout (MaxAge <= 0)")
		}
	})

	t.Run("Logout with Bearer token", func(t *testing.T) {
		validToken := svc.IssueSession(testUser.ID, 24*time.Hour)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.LogoutHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on logout, got %d", rec.Code)
		}
	})

	t.Run("Logout method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()

		handler.LogoutHandler(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", rec.Code)
		}
	})

	t.Run("MeHandler returns user profile for authenticated user", func(t *testing.T) {
		ctx := auth.WithUserID(context.Background(), testUser.ID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.MeHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK from MeHandler, got %d", rec.Code)
		}

		var resp struct {
			ID         string `json:"id"`
			Email      string `json:"email"`
			Role       string `json:"role"`
			QuotaBytes int64  `json:"quota_bytes"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode MeHandler response: %v", err)
		}

		if resp.ID != testUser.ID.String() {
			t.Errorf("expected ID %s, got %s", testUser.ID, resp.ID)
		}
		if resp.Email != testUser.Email {
			t.Errorf("expected email %s, got %s", testUser.Email, resp.Email)
		}
		if resp.QuotaBytes != testUser.QuotaBytes {
			t.Errorf("expected quota_bytes %d, got %d", testUser.QuotaBytes, resp.QuotaBytes)
		}
	})

	t.Run("MeHandler unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()

		handler.MeHandler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("MeHandler method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()

		handler.MeHandler(rec, req)

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

	t.Run("RequireSameOrigin blocks foreign origin on mutating request with 403", func(t *testing.T) {
		nextCalled := false
		protected := auth.RequireSameOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", nil)
		req.Host = "cloud.example.com"
		req.Header.Set("Origin", "http://evil.com")
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for foreign origin, got %d", rec.Code)
		}
		if nextCalled {
			t.Error("expected inner handler not to be executed on CSRF mismatch")
		}
	})

	t.Run("RequireSameOrigin allows matching origin", func(t *testing.T) {
		nextCalled := false
		protected := auth.RequireSameOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", nil)
		req.Host = "cloud.example.com"
		req.Header.Set("Origin", "http://cloud.example.com")
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !nextCalled {
			t.Errorf("expected 200 OK and next called for matching origin, got %d", rec.Code)
		}
	})

	t.Run("RequireSameOrigin allows matching origin via X-Forwarded-Host", func(t *testing.T) {
		nextCalled := false
		protected := auth.RequireSameOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", nil)
		req.Host = "internal:8080"
		req.Header.Set("X-Forwarded-Host", "cloud.example.com")
		req.Header.Set("Origin", "https://cloud.example.com")
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !nextCalled {
			t.Errorf("expected 200 OK and next called for X-Forwarded-Host matching origin, got %d", rec.Code)
		}
	})

	t.Run("RequireSameOrigin allows request without Origin header", func(t *testing.T) {
		nextCalled := false
		protected := auth.RequireSameOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", nil)
		req.Host = "cloud.example.com"
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !nextCalled {
			t.Errorf("expected 200 OK for request without Origin, got %d", rec.Code)
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

	t.Run("MeHandler JSON output includes used_bytes and quota_bytes even when used_bytes is 0", func(t *testing.T) {
		zeroUser := auth.User{
			ID:         uuid.New(),
			Email:      "zero@example.com",
			Role:       "user",
			QuotaBytes: 5000,
			UsedBytes:  0,
		}
		svc.AddUser(zeroUser, testPassword)
		ctx := auth.WithUserID(context.Background(), zeroUser.ID)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.MeHandler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		rawBody := rec.Body.String()
		if !strings.Contains(rawBody, `"used_bytes":0`) && !strings.Contains(rawBody, `"used_bytes": 0`) {
			t.Errorf("expected JSON to contain 'used_bytes': 0 (no omitempty), got: %s", rawBody)
		}
		if !strings.Contains(rawBody, `"quota_bytes":5000`) && !strings.Contains(rawBody, `"quota_bytes": 5000`) {
			t.Errorf("expected JSON to contain 'quota_bytes': 5000 (no omitempty), got: %s", rawBody)
		}
	})
}

