package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func TestFileDownload_IDORUniform404(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)

	rr := uploadTestFile(t, fh, userA, "secret.txt", []byte("user A secret"), "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rr.Code)
	}
	meta := decodeFileMeta(t, rr)

	t.Run("another user's file returns 404", func(t *testing.T) {
		dlRR := downloadFileRequest(fh, userB, meta.ID)
		if dlRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 for foreign file download, got %d", dlRR.Code)
		}
	})

	t.Run("non-existent UUID returns identical 404", func(t *testing.T) {
		dlRR := downloadFileRequest(fh, userB, uuid.New().String())
		if dlRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 for non-existent file download, got %d", dlRR.Code)
		}
	})

	t.Run("binary on disk without DB record is never served", func(t *testing.T) {
		orphanID := uuid.New().String()
		if _, _, err := engine.Save(orphanID, bytes.NewReader([]byte("orphan binary")), 1024); err != nil {
			t.Fatalf("failed to place orphan binary on disk: %v", err)
		}
		dlRR := downloadFileRequest(fh, userA, orphanID)
		if dlRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 for file present on disk but absent in DB, got %d", dlRR.Code)
		}
	})
}

func TestFileHandler_DownloadHandler_BranchCoverage(t *testing.T) {
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	userID := uuid.New()

	t.Run("DownloadHandler empty file ID returns 400", func(t *testing.T) {
		fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
		rr := downloadFileRequest(fh, userID, "")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty file ID, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler non-UUID file ID returns 400 Bad Request", func(t *testing.T) {
		fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
		rr := downloadFileRequest(fh, userID, "not-a-valid-uuid")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for non-UUID file ID, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler pool is nil returns 404", func(t *testing.T) {
		fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
		rr := downloadFileRequest(fh, userID, uuid.New().String())
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 when pool is nil, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler missing binary file on disk returns 404", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		fileID := uuid.New()
		missingStoragePath := "/nonexistent/path/to/missing.bin"
		_, err := pool.Exec(context.Background(),
			`INSERT INTO files (id, user_id, filename, size_bytes, sha256_hash, storage_path)
			 VALUES ($1, $2, 'missing.bin', 10, repeat('a', 64), $3)`,
			fileID, testUser, missingStoragePath)
		if err != nil {
			t.Fatalf("failed to insert missing file row: %v", err)
		}

		rr := downloadFileRequest(fh, testUser, fileID.String())
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 when file is missing from disk, got %d", rr.Code)
		}
	})
}

func TestFileDownload_ContentDispositionFormat(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)

	t.Run("Cyrillic filename contains filename* UTF-8 RFC 5987 and ASCII fallback", func(t *testing.T) {
		cyrillicName := "отчёт.pdf"
		rr := uploadTestFile(t, fh, userID, cyrillicName, []byte("cyrillic content"), "")
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", rr.Code)
		}
		meta := decodeFileMeta(t, rr)

		dlRR := downloadFileRequest(fh, userID, meta.ID)
		if dlRR.Code != http.StatusOK {
			t.Fatalf("expected 200 OK download, got %d", dlRR.Code)
		}

		cd := dlRR.Header().Get("Content-Disposition")
		if !strings.Contains(cd, "filename*=UTF-8''") {
			t.Errorf("expected Content-Disposition to contain filename*=UTF-8'', got %q", cd)
		}
		expectedEscaped := url.PathEscape(cyrillicName)
		if !strings.Contains(cd, expectedEscaped) {
			t.Errorf("expected Content-Disposition to contain percent-encoded name %q, got %q", expectedEscaped, cd)
		}
		if !strings.Contains(cd, "filename=") {
			t.Errorf("expected Content-Disposition to contain ASCII fallback filename, got %q", cd)
		}
	})

	t.Run("ASCII filename uses filename=\"report.pdf\"", func(t *testing.T) {
		asciiName := "report.pdf"
		rr := uploadTestFile(t, fh, userID, asciiName, []byte("ascii content"), "")
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", rr.Code)
		}
		meta := decodeFileMeta(t, rr)

		dlRR := downloadFileRequest(fh, userID, meta.ID)
		if dlRR.Code != http.StatusOK {
			t.Fatalf("expected 200 OK download, got %d", dlRR.Code)
		}

		cd := dlRR.Header().Get("Content-Disposition")
		expected := `attachment; filename="report.pdf"`
		if cd != expected {
			t.Errorf("expected Content-Disposition %q, got %q", expected, cd)
		}
	})

	t.Run("Filename with CR/LF and quotes is sanitized against header injection", func(t *testing.T) {
		dirtyName := "report\r\nX-Injected-Header: evil\n\"quote.pdf"
		fileID := uuid.New()
		storagePath := filepath.Join(tempDir, fileID.String())
		if err := os.WriteFile(storagePath, []byte("dirty content"), 0644); err != nil {
			t.Fatalf("failed to write dummy storage file: %v", err)
		}

		_, err := pool.Exec(context.Background(),
			`INSERT INTO files (id, user_id, filename, storage_path, size_bytes, mime_type, sha256_hash) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			fileID, userID, dirtyName, storagePath, 13, "application/octet-stream", "dummyhash123")
		if err != nil {
			t.Fatalf("failed to insert test file record with dirty filename: %v", err)
		}

		dlRR := downloadFileRequest(fh, userID, fileID.String())
		if dlRR.Code != http.StatusOK {
			t.Fatalf("expected 200 OK download, got %d", dlRR.Code)
		}

		if dlRR.Header().Get("X-Injected-Header") != "" {
			t.Error("header injection detected: X-Injected-Header was set")
		}

		cd := dlRR.Header().Get("Content-Disposition")
		if strings.Contains(cd, "\r") || strings.Contains(cd, "\n") || strings.Contains(cd, `\"`) {
			t.Errorf("Content-Disposition contains unsanitized CR, LF, or quotes: %q", cd)
		}
	})
}
