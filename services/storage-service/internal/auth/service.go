package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized session")
	ErrUserNotFound       = errors.New("user not found")
)

type User struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	QuotaBytes int64     `json:"quota_bytes,omitempty"`
	UsedBytes  int64     `json:"used_bytes,omitempty"`
	Role       string    `json:"role"`
	IsActive   bool      `json:"is_active,omitempty"`
}

type Service interface {
	Login(ctx context.Context, email, password, userAgent, clientIP string) (token string, user *User, err error)
	Logout(ctx context.Context, token string) error
	ValidateSession(ctx context.Context, token string) (uuid.UUID, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateSessionToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(bytes)
	hashBytes := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(hashBytes[:])
	return token, hash, nil
}

func hashToken(token string) string {
	hashBytes := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hashBytes[:])
}

// SeedAdminUser checks if an admin user exists; if missing, inserts default admin account with bcrypt hashed password.
func SeedAdminUser(ctx context.Context, pool *pgxpool.Pool, adminEmail, adminPassword string) error {
	if adminEmail == "" || adminPassword == "" {
		log.Println("[WARN] ADMIN_EMAIL or ADMIN_PASSWORD not configured. Skipping admin account seeding.")
		return nil
	}

	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE role = 'admin' OR email = $1)`, adminEmail).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	hashedPassword, err := HashPassword(adminPassword)
	if err != nil {
		return err
	}

	adminID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	quotaBytes := int64(50 * 1024 * 1024 * 1024)

	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, quota_bytes, used_bytes, role, is_active)
		VALUES ($1, $2, $3, $4, 0, 'admin', true)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			quota_bytes = EXCLUDED.quota_bytes,
			role = EXCLUDED.role,
			is_active = EXCLUDED.is_active
	`, adminID, adminEmail, hashedPassword, quotaBytes)

	return err
}

type DBAuthService struct {
	pool            *pgxpool.Pool
	sessionDuration time.Duration
}

func NewDBAuthService(pool *pgxpool.Pool, sessionDuration time.Duration) *DBAuthService {
	if sessionDuration <= 0 {
		sessionDuration = 24 * time.Hour
	}
	return &DBAuthService{
		pool:            pool,
		sessionDuration: sessionDuration,
	}
}

func (s *DBAuthService) Login(ctx context.Context, email, password, userAgent, clientIP string) (string, *User, error) {
	var user User
	var passwordHash string
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, is_active, quota_bytes, used_bytes FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &passwordHash, &user.Role, &user.IsActive, &user.QuotaBytes, &user.UsedBytes)

	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	if !user.IsActive || !CheckPasswordHash(password, passwordHash) {
		return "", nil, ErrInvalidCredentials
	}

	token, tokenHash, err := GenerateSessionToken()
	if err != nil {
		return "", nil, err
	}

	sessionID := uuid.New()
	expiresAt := time.Now().Add(s.sessionDuration)

	_, err = s.pool.Exec(ctx,
		`INSERT INTO user_sessions (id, user_id, token_hash, expires_at, user_agent, client_ip) VALUES ($1, $2, $3, $4, $5, $6)`,
		sessionID, user.ID, tokenHash, expiresAt, userAgent, clientIP,
	)
	if err != nil {
		return "", nil, err
	}

	return token, &user, nil
}

func (s *DBAuthService) Logout(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	_, err := s.pool.Exec(ctx, `DELETE FROM user_sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *DBAuthService) ValidateSession(ctx context.Context, token string) (uuid.UUID, error) {
	tokenHash := hashToken(token)
	var userID uuid.UUID
	var expiresAt time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT user_id, expires_at FROM user_sessions WHERE token_hash = $1`,
		tokenHash,
	).Scan(&userID, &expiresAt)

	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}

	if time.Now().After(expiresAt) {
		_, _ = s.pool.Exec(ctx, `DELETE FROM user_sessions WHERE token_hash = $1`, tokenHash)
		return uuid.Nil, ErrUnauthorized
	}

	return userID, nil
}

func (s *DBAuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var user User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, role, is_active, quota_bytes, used_bytes FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Role, &user.IsActive, &user.QuotaBytes, &user.UsedBytes)

	if err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (s *DBAuthService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	commandTag, err := s.pool.Exec(ctx, `DELETE FROM user_sessions WHERE expires_at < CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

func (s *DBAuthService) StartCleanupWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				_, _ = s.CleanupExpiredSessions(ctx)
			}
		}
	}()
}
