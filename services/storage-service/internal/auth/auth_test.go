package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

func TestPasswordHashing(t *testing.T) {
	password := "SecretPassword123!"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got: %v", err)
	}

	if hash == "" || hash == password {
		t.Fatalf("hash should not be empty or equal to plain password")
	}

	if !auth.CheckPasswordHash(password, hash) {
		t.Errorf("expected CheckPasswordHash to return true for matching password")
	}

	if auth.CheckPasswordHash("WrongPassword", hash) {
		t.Errorf("expected CheckPasswordHash to return false for wrong password")
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Initially no user_id in context
	_, ok := auth.GetUserIDFromContext(ctx)
	if ok {
		t.Error("expected GetUserIDFromContext to return false for empty context")
	}

	userID := uuid.New()
	ctxWithUser := auth.WithUserID(ctx, userID)

	retrievedID, ok := auth.GetUserIDFromContext(ctxWithUser)
	if !ok {
		t.Fatal("expected GetUserIDFromContext to return true for context with user_id")
	}

	if retrievedID != userID {
		t.Errorf("expected user_id %s, got %s", userID, retrievedID)
	}
}

func TestSessionTokenGeneration(t *testing.T) {
	token, hash, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("expected no error generating session token, got: %v", err)
	}

	if len(token) < 32 {
		t.Errorf("expected token length >= 32, got %d", len(token))
	}

	if len(hash) != 64 { // SHA256 hex string is 64 chars
		t.Errorf("expected token hash length 64, got %d", len(hash))
	}
}

func TestStartCleanupWorker(t *testing.T) {
	svc := auth.NewDBAuthService(nil, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so ticker loop exits right away
	svc.StartCleanupWorker(ctx, 1*time.Second)
	time.Sleep(10 * time.Millisecond)
}
