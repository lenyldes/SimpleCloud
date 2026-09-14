package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
		rr := uploadTestFile(t, fh, userID, "test.txt", []byte("hello"), "")
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler user record not found in DB returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		nonExistentUserID := uuid.New()
		rr := uploadTestFile(t, fh, nonExistentUserID, "test.txt", []byte("hello"), "")
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
		rr := uploadTestFile(t, fh, overQuotaUserID, "over.txt", []byte("this content is much longer than 10 bytes remaining"), "")
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

func TestFileUpload_Validation_FilenameLimits(t *testing.T) {
	pool := setupTestPool(t)

	t.Run("filename longer than 255 characters returns 400", func(t *testing.T) {
		tempDir := t.TempDir()
		fh := handler.NewFileHandler(storage.NewDiskEngine(tempDir), pool, 10*1024*1024)
		userID := createTestUser(t, pool, 10*1024*1024)

		longName := strings.Repeat("a", 256) + ".txt"
		rr := uploadTestFile(t, fh, userID, longName, []byte("content"), "")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for filename > 255 chars, got %d, body: %s", rr.Code, rr.Body.String())
		}
		if got := countUserFiles(t, pool, userID); got != 0 {
			t.Errorf("expected 0 files in DB after rejected upload, got %d", got)
		}
		if got := countRegularFiles(t, tempDir); got != 0 {
			t.Errorf("expected 0 files on disk after rejected upload, got %d", got)
		}
	})

	t.Run("empty or whitespace filename returns 400", func(t *testing.T) {
		tempDir := t.TempDir()
		fh := handler.NewFileHandler(storage.NewDiskEngine(tempDir), pool, 10*1024*1024)
		userID := createTestUser(t, pool, 10*1024*1024)

		rr := uploadTestFile(t, fh, userID, "   ", []byte("content"), "")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for whitespace filename, got %d, body: %s", rr.Code, rr.Body.String())
		}
		if got := countUserFiles(t, pool, userID); got != 0 {
			t.Errorf("expected 0 files in DB after whitespace filename, got %d", got)
		}
		if got := countRegularFiles(t, tempDir); got != 0 {
			t.Errorf("expected 0 files on disk after whitespace filename, got %d", got)
		}
	})
}

func TestFolderCreation_Validation_NullBytes(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	folderH := handler.NewFolderHandler(pool, engine)
	userID := createTestUser(t, pool, 10*1024*1024)

	t.Run("folder name containing null byte returns 400", func(t *testing.T) {
		rr := createFolderRequest(folderH, userID, map[string]interface{}{"name": "bad\x00folder"})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for folder name with null byte, got %d, body: %s", rr.Code, rr.Body.String())
		}
		var count int
		_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM folders WHERE user_id = $1", userID).Scan(&count)
		if count != 0 {
			t.Errorf("expected 0 folders in DB after rejected creation, got %d", count)
		}
	})
}

type streamPauseReader struct {
	r       io.Reader
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *streamPauseReader) Read(p []byte) (int, error) {
	s.once.Do(func() {
		close(s.started)
		<-s.release
	})
	return s.r.Read(p)
}

func TestFileUpload_TwoPhaseQuota_NoDBLockAndOverdrawRollback(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 1000)
	userID := createTestUser(t, pool, 1000)

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", "phase_test.bin")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(bytes.Repeat([]byte("Z"), 600))
	_ = writer.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	pauseReader := &streamPauseReader{r: &b, started: started, release: release}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", pauseReader)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(auth.WithUserID(req.Context(), userID))

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		done <- rr
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for upload stream to start")
	}

	// 1. Verify user row is NOT locked in PostgreSQL while streaming
	lockCtx, lockCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer lockCancel()
	_, lockErr := pool.Exec(lockCtx, "SELECT id FROM users WHERE id = $1 FOR UPDATE", userID)
	if lockErr != nil {
		t.Errorf("expected no DB row lock during Phase 1 streaming, but lock query failed: %v", lockErr)
	}

	// 2. Consume quota concurrently before Phase 2 commit
	updateCtx, updateCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer updateCancel()
	_, err = pool.Exec(updateCtx, "UPDATE users SET used_bytes = 600 WHERE id = $1", userID)
	if err != nil {
		t.Errorf("expected concurrent quota update to succeed, but got error: %v", err)
	}

	close(release)
	rr := <-done

	// 3. Phase 2 must detect quota exceeded, return 413, rollback, and delete orphan file
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 on concurrent quota exhaustion in Phase 2, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if got := getUserUsedBytes(t, pool, userID); got != 600 {
		t.Errorf("expected used_bytes to remain 600 after rollback, got %d", got)
	}
	if got := countUserFiles(t, pool, userID); got != 0 {
		t.Errorf("expected 0 files in DB after rollback, got %d", got)
	}
	if got := countRegularFiles(t, tempDir); got != 0 {
		t.Errorf("expected 0 regular files on disk (orphan cleanup), got %d", got)
	}
}

func TestFileUpload_TwoPhaseQuota_ConcurrentRace(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 1000)
	userID := createTestUser(t, pool, 1000)

	var wg sync.WaitGroup
	results := make([]*httptest.ResponseRecorder, 2)
	payload := bytes.Repeat([]byte("W"), 600)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = uploadTestFile(t, fh, userID, fmt.Sprintf("race_%d.bin", idx), payload, "")
		}(i)
	}
	wg.Wait()

	has201 := results[0].Code == http.StatusCreated || results[1].Code == http.StatusCreated
	has413 := results[0].Code == http.StatusRequestEntityTooLarge || results[1].Code == http.StatusRequestEntityTooLarge
	if !has201 || !has413 {
		t.Fatalf("expected one 201 and one 413, got: %d, %d", results[0].Code, results[1].Code)
	}
	if got := countUserFiles(t, pool, userID); got != 1 {
		t.Errorf("expected 1 file in DB, got %d", got)
	}
	if got := countRegularFiles(t, tempDir); got != 1 {
		t.Errorf("expected 1 file on disk, got %d", got)
	}
	if got := getUserUsedBytes(t, pool, userID); got != 600 {
		t.Errorf("expected used_bytes = 600, got %d", got)
	}
}

func TestFileUpload_OrphanFileCleanup_LoggedOnRemoveError(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
	userID := createTestUser(t, pool, 10*1024*1024)

	var logBuf bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origOutput)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", "fail_cleanup.bin")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(bytes.Repeat([]byte("K"), 64*1024))
	_ = writer.Close()

	stopWatcher := make(chan struct{})
	defer close(stopWatcher)
	go func() {
		for {
			select {
			case <-stopWatcher:
				return
			default:
				_ = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
					if info != nil && info.IsDir() && path != tempDir {
						_ = os.Chmod(path, 0555)
						cancel()
					}
					return nil
				})
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	t.Cleanup(func() {
		_ = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if info != nil && info.IsDir() {
				_ = os.Chmod(path, 0777)
			}
			return nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(auth.WithUserID(ctx, userID))

	rr := httptest.NewRecorder()
	fh.UploadHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on canceled context during commit, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(logBuf.String(), "failed to clean up orphan storage file") {
		t.Errorf("expected log output containing 'failed to clean up orphan storage file', got: %q", logBuf.String())
	}
}
