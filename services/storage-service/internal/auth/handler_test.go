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
	svc := auth.NewMockAuthService()
	handler := auth.NewAuthHandler(svc)

	t.Run("Login success sets HttpOnly cookie", func(t *testing.T) {
		loginPayload := map[string]string{
			"email":    "admin@simplecloud.local",
			"password": "adminpassword123",
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
			"email":    "admin@simplecloud.local",
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
		validToken := svc.CreateValidSessionToken(uuid.New(), 24*time.Hour)

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
		validToken := svc.CreateValidSessionToken(uuid.New(), 24*time.Hour)

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
		validToken := svc.CreateValidSessionToken(uuid.New(), 24*time.Hour)

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
		validToken := svc.CreateValidSessionToken(uuid.New(), 24*time.Hour)

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
		userID := uuid.New()
		ctx := auth.WithUserID(context.Background(), userID)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.MeHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK from MeHandler, got %d", rec.Code)
		}

		var resp struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode MeHandler response: %v", err)
		}

		if resp.ID != userID.String() {
			t.Errorf("expected ID %s, got %s", userID, resp.ID)
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

}
