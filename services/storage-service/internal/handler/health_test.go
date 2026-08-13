package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
)

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(handler.HealthHandler)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rr.Code)
	}

	expectedContentType := "application/json"
	if ct := rr.Header().Get("Content-Type"); ct != expectedContentType {
		t.Errorf("expected Content-Type %s, got %s", expectedContentType, ct)
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
}
