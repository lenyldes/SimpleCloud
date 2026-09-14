package handler_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func TestFileHandler_UploadHandler_BranchCoverage(t *testing.T) {
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	userID := uuid.New()

	t.Run("UploadHandler pool is nil returns 500", func(t *testing.T) {
		fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
		req := buildUploadRequest(t, userID, "test.txt", []byte("hello"), "")
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler user record not found in DB returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		nonExistentUserID := uuid.New()
		req := buildUploadRequest(t, nonExistentUserID, "test.txt", []byte("hello"), "")
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 for missing user record, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler user used_bytes near quota_bytes limits remaining quota -> 413", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 1000)
		overQuotaUserID := createTestUser(t, pool, 1000)
		_, err := pool.Exec(context.Background(), `UPDATE users SET used_bytes = 990 WHERE id = $1`, overQuotaUserID)
		if err != nil {
			t.Fatalf("failed to update used_bytes: %v", err)
		}
		req := buildUploadRequest(t, overQuotaUserID, "over.txt", []byte("this content is much longer than 10 bytes remaining"), "")
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected 413 when file exceeds remaining quota, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler non-numeric Content-Length header is ignored", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		req := buildUploadRequest(t, testUser, "valid.txt", []byte("valid content"), "")
		req.Header.Set("Content-Length", "invalid-number")
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusCreated {
			t.Errorf("expected 201 when Content-Length is non-numeric string, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler invalid multipart body returns 400", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", strings.NewReader("not a multipart body"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary123")
		req = req.WithContext(auth.WithUserID(req.Context(), testUser))
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid multipart form, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler missing file field in form returns 400", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("other_field", "value")
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(auth.WithUserID(req.Context(), testUser))
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing file field, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler non-UUID string folder_id returns 400", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		rr := uploadTestFile(t, fh, testUser, "nonuuid_folder.txt", []byte("content"), "invalid-folder-uuid")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for non-UUID string folder_id, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler non-existent folder_id returns 404", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		missingFolder := uuid.New().String()
		rr := uploadTestFile(t, fh, testUser, "missing_folder.txt", []byte("content"), missingFolder)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for non-existent user folder_id, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler canceled context on tx begin returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		req := buildUploadRequest(t, testUser, "canceled.txt", []byte("content"), "")
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 when transaction start fails on canceled context, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler canceled context on folder verification returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		folderID := createTestFolder(t, pool, testUser)

		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		_ = writer.WriteField("folder_id", folderID.String())
		part, _ := writer.CreateFormFile("file", "test.txt")
		_, _ = part.Write([]byte("content"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", nil)
		ctx, cancel := context.WithCancel(req.Context())
		req = req.WithContext(auth.WithUserID(ctx, testUser))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Body = &cancelOnReadCloser{r: &b, cancel: cancel}

		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 when folder verification fails on canceled context, got %d, body: %s", rr.Code, rr.Body.String())
		}
	})
}

type cancelOnReadCloser struct {
	r      io.Reader
	cancel context.CancelFunc
}

func (c *cancelOnReadCloser) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if c.cancel != nil {
		c.cancel()
	}
	return n, err
}

func (c *cancelOnReadCloser) Close() error {
	return nil
}
