package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/database"
)

func TestDBAuthService_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connStr := "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"
	pool, err := database.InitDB(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping DBAuthService integration test; postgres database not accessible: %v", err)
	}
	defer pool.Close()

	// Clean up any stale admin/test user records to avoid primary key conflict
	_, _ = pool.Exec(ctx, `DELETE FROM user_sessions`)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000001' OR email = 'admin@simplecloud.local'`)

	// 1. Test SeedAdminUser
	err = auth.SeedAdminUser(ctx, pool, "admin@simplecloud.local", "adminpassword123")
	if err != nil {
		t.Fatalf("failed to seed admin user: %v", err)
	}

	// Re-seeding when already existing should return nil
	err = auth.SeedAdminUser(ctx, pool, "admin@simplecloud.local", "adminpassword123")
	if err != nil {
		t.Fatalf("re-seeding admin user failed: %v", err)
	}

	dbAuth := auth.NewDBAuthService(pool, 1*time.Hour)

	// 2. Test Login
	token, user, err := dbAuth.Login(ctx, "admin@simplecloud.local", "adminpassword123", "GoTest", "127.0.0.1")
	if err != nil {
		t.Fatalf("expected successful login, got: %v", err)
	}
	if token == "" || user == nil {
		t.Fatal("expected non-empty token and user pointer")
	}

	// Login invalid password
	_, _, err = dbAuth.Login(ctx, "admin@simplecloud.local", "WrongPass", "GoTest", "127.0.0.1")
	if err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for wrong pass, got: %v", err)
	}

	// Login non-existent email
	_, _, err = dbAuth.Login(ctx, "nonexistent@simplecloud.local", "AdminPass123!", "GoTest", "127.0.0.1")
	if err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for non-existent email, got: %v", err)
	}

	// 3. Test ValidateSession
	userID, err := dbAuth.ValidateSession(ctx, token)
	if err != nil {
		t.Fatalf("expected valid session, got: %v", err)
	}
	if userID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, userID)
	}

	// Validate non-existent token
	_, err = dbAuth.ValidateSession(ctx, "invalid_token_string")
	if err != auth.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized for invalid token, got: %v", err)
	}

	// 4. Test GetUserByID
	fetchedUser, err := dbAuth.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to get user by ID: %v", err)
	}
	if fetchedUser.Email != "admin@simplecloud.local" {
		t.Errorf("expected email admin@simplecloud.local, got %s", fetchedUser.Email)
	}

	_, err = dbAuth.GetUserByID(ctx, uuid.New())
	if err != auth.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound for non-existent UUID, got: %v", err)
	}

	// 5. Test Logout
	err = dbAuth.Logout(ctx, token)
	if err != nil {
		t.Fatalf("failed to logout: %v", err)
	}

	// After logout, session should be invalid
	_, err = dbAuth.ValidateSession(ctx, token)
	if err != auth.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized after logout, got: %v", err)
	}
}
