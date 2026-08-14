package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/database"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
)

const testDBConnString = "postgres://simplecloud_user:simplecloud_dev_password@127.0.0.1:5432/simplecloud?sslmode=disable"

// setupTestPool connects to the local PostgreSQL (running migrations) or skips
// the integration test when the database is not reachable (auth_db_test.go pattern).
func setupTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.InitDB(ctx, testDBConnString)
	if err != nil {
		t.Skipf("Skipping integration test; postgres database not accessible: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// createTestUser inserts a dedicated user row with the given quota and schedules
// cascade cleanup (files/folders/sessions are removed via ON DELETE CASCADE).
func createTestUser(t *testing.T, pool *pgxpool.Pool, quotaBytes int64) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := fmt.Sprintf("test-%s@simplecloud.test", userID.String())
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, quota_bytes, used_bytes) VALUES ($1, $2, $3, 0)`,
		userID, email, quotaBytes)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

// createTestFolder inserts a folder row directly (used when a file upload needs a
// valid folder_id FK target without going through FolderHandler).
func createTestFolder(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) uuid.UUID {
	t.Helper()
	folderID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO folders (id, user_id, parent_id, name) VALUES ($1, $2, NULL, $3)`,
		folderID, userID, "test-folder-"+folderID.String())
	if err != nil {
		t.Fatalf("failed to create test folder row: %v", err)
	}
	return folderID
}

func getUserUsedBytes(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int64 {
	t.Helper()
	var used int64
	err := pool.QueryRow(context.Background(),
		`SELECT used_bytes FROM users WHERE id = $1`, userID).Scan(&used)
	if err != nil {
		t.Fatalf("failed to read used_bytes for user %s: %v", userID, err)
	}
	return used
}

func countUserFiles(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM files WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count files for user %s: %v", userID, err)
	}
	return count
}

func countUserFolders(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM folders WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count folders for user %s: %v", userID, err)
	}
	return count
}

func buildUploadRequest(t *testing.T, userID uuid.UUID, filename string, content []byte, folderID string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if folderID != "" {
		if err := writer.WriteField("folder_id", folderID); err != nil {
			t.Fatalf("failed to write folder_id form field: %v", err)
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write form file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	if err != nil {
		t.Fatalf("failed to create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req.WithContext(auth.WithUserID(req.Context(), userID))
}

func uploadTestFile(t *testing.T, fh *handler.FileHandler, userID uuid.UUID, filename string, content []byte, folderID string) *httptest.ResponseRecorder {
	t.Helper()
	req := buildUploadRequest(t, userID, filename, content, folderID)
	rr := httptest.NewRecorder()
	fh.UploadHandler(rr, req)
	return rr
}

// countRegularFiles walks the storage root and counts regular files; used to
// verify disk cleanup leaves no binary shards or temp fragments behind.
func countRegularFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type().IsRegular() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk storage dir %s: %v", root, err)
	}
	return count
}
