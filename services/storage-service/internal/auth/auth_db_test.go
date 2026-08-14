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

	// 1. Test SeedAdminUser with empty arguments (should skip seeding and NOT create admin)
	err = auth.SeedAdminUser(ctx, pool, "", "")
	if err != nil {
		t.Fatalf("expected nil error when seeding with empty args, got: %v", err)
	}
	var emptySeedCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin' OR email = 'admin@simplecloud.local'`).Scan(&emptySeedCount)
	if err != nil {
		t.Fatalf("failed to query admin user count after empty seed: %v", err)
	}
	if emptySeedCount != 0 {
		t.Errorf("expected 0 admin users created when empty args supplied, got %d", emptySeedCount)
	}

	// Test SeedAdminUser with missing password (should skip seeding and NOT create admin)
	err = auth.SeedAdminUser(ctx, pool, "admin@simplecloud.local", "")
	if err != nil {
		t.Fatalf("expected nil error when seeding with empty password, got: %v", err)
	}
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin' OR email = 'admin@simplecloud.local'`).Scan(&emptySeedCount)
	if err != nil {
		t.Fatalf("failed to query admin user count after empty password seed: %v", err)
	}
	if emptySeedCount != 0 {
		t.Errorf("expected 0 admin users created when empty password supplied, got %d", emptySeedCount)
	}

	// Test SeedAdminUser with valid arguments (should seed admin user successfully)
	err = auth.SeedAdminUser(ctx, pool, "admin@simplecloud.local", "adminpassword123")
	if err != nil {
		t.Fatalf("failed to seed admin user with valid args: %v", err)
	}
	var validSeedCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin' OR email = 'admin@simplecloud.local'`).Scan(&validSeedCount)
	if err != nil {
		t.Fatalf("failed to query admin user count after valid seed: %v", err)
	}
	if validSeedCount != 1 {
		t.Errorf("expected 1 admin user created after valid seed, got %d", validSeedCount)
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

func TestDBAuthService_CleanupExpiredSessions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connStr := "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"
	pool, err := database.InitDB(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping DBAuthService cleanup test; postgres database not accessible: %v", err)
	}
	defer pool.Close()

	// Clean up stale session & user data
	_, _ = pool.Exec(ctx, `DELETE FROM user_sessions`)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000001' OR email = 'admin@simplecloud.local'`)

	err = auth.SeedAdminUser(ctx, pool, "admin@simplecloud.local", "adminpassword123")
	if err != nil {
		t.Fatalf("failed to seed admin user: %v", err)
	}

	dbAuth := auth.NewDBAuthService(pool, 1*time.Hour)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Insert expired session (expired 2 hours ago)
	expiredSessionID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO user_sessions (id, user_id, token_hash, expires_at, user_agent, client_ip) VALUES ($1, $2, $3, $4, $5, $6)`,
		expiredSessionID, userID, "hash_expired_session_123", time.Now().Add(-2*time.Hour), "TestAgent", "127.0.0.1",
	)
	if err != nil {
		t.Fatalf("failed to insert expired session: %v", err)
	}

	// Insert valid active session (expires in 2 hours)
	validSessionID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO user_sessions (id, user_id, token_hash, expires_at, user_agent, client_ip) VALUES ($1, $2, $3, $4, $5, $6)`,
		validSessionID, userID, "hash_valid_session_456", time.Now().Add(2*time.Hour), "TestAgent", "127.0.0.1",
	)
	if err != nil {
		t.Fatalf("failed to insert valid session: %v", err)
	}

	// Call CleanupExpiredSessions method on dbAuth
	purgedCount, err := dbAuth.CleanupExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("expected no error during CleanupExpiredSessions, got: %v", err)
	}
	if purgedCount != 1 {
		t.Errorf("expected 1 session purged, got %d", purgedCount)
	}

	// Verify DB state: expired session should be deleted, valid session present
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_sessions WHERE id = $1`, expiredSessionID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query expired session: %v", err)
	}
	if count != 0 {
		t.Errorf("expected expired session to be deleted from DB, but found %d rows", count)
	}

	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_sessions WHERE id = $1`, validSessionID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query valid session: %v", err)
	}
	if count != 1 {
		t.Errorf("expected valid session to remain in DB, but found %d rows", count)
	}
}

func TestDBAuthService_StartCleanupWorker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connStr := "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"
	pool, err := database.InitDB(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping DBAuthService cleanup worker test; postgres database not accessible: %v", err)
	}
	defer pool.Close()

	dbAuth := auth.NewDBAuthService(pool, 1*time.Hour)
	workerCtx, workerCancel := context.WithCancel(ctx)
	dbAuth.StartCleanupWorker(workerCtx, 10*time.Millisecond)
	time.Sleep(35 * time.Millisecond)
	workerCancel()
}

func TestTimingSafeLogin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connStr := "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"
	pool, err := database.InitDB(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping integration test; postgres database not accessible: %v", err)
	}
	defer pool.Close()

	dbAuth := auth.NewDBAuthService(pool, 1*time.Hour)

	start := time.Now()
	_, _, err = dbAuth.Login(ctx, "nonexistent-timing-test@simplecloud.local", "TestPassword123!", "GoTest", "127.0.0.1")
	elapsed := time.Since(start)

	if err != auth.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	if elapsed < 2*time.Millisecond {
		t.Errorf("expected timing-safe login with non-existent email to execute bcrypt check (elapsed >= 2ms), took %v", elapsed)
	}
}
