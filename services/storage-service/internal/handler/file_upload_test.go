package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func TestFileUpload_PersistsMetadataToDB(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)
	content := []byte("persistent metadata payload")
	expectedSHA := sha256.Sum256(content)

	rr := uploadTestFile(t, fh, userID, "persist.txt", content, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d, body: %s", rr.Code, rr.Body.String())
	}
	meta := decodeFileMeta(t, rr)
	if meta.Filename != "persist.txt" || meta.Size != int64(len(content)) {
		t.Errorf("unexpected response metadata: %+v", meta)
	}
	if hex.EncodeToString(expectedSHA[:]) != meta.SHA256 {
		t.Errorf("expected sha256 %s, got %s", hex.EncodeToString(expectedSHA[:]), meta.SHA256)
	}

	var dbFilename, dbSHA, dbStoragePath string
	var dbSize int64
	var dbUserID uuid.UUID
	var dbFolderID *uuid.UUID
	err := pool.QueryRow(context.Background(),
		`SELECT user_id, filename, size_bytes, sha256_hash, storage_path, folder_id FROM files WHERE id = $1`,
		meta.ID).Scan(&dbUserID, &dbFilename, &dbSize, &dbSHA, &dbStoragePath, &dbFolderID)
	if err != nil {
		t.Fatalf("expected files row for uploaded file, query failed: %v", err)
	}
	if dbUserID != userID {
		t.Errorf("expected user_id %s in files row, got %s", userID, dbUserID)
	}
	if dbFilename != "persist.txt" {
		t.Errorf("expected filename persist.txt, got %q", dbFilename)
	}
	if dbSize != int64(len(content)) {
		t.Errorf("expected size_bytes %d, got %d", len(content), dbSize)
	}
	if dbSHA != hex.EncodeToString(expectedSHA[:]) {
		t.Errorf("expected sha256_hash %s, got %s", hex.EncodeToString(expectedSHA[:]), dbSHA)
	}
	if dbStoragePath == "" {
		t.Error("expected non-empty storage_path in files row")
	}
	if dbFolderID != nil {
		t.Errorf("expected NULL folder_id for root upload, got %v", dbFolderID)
	}

	listRR := listFilesRequest(fh, userID, "/api/v1/files")
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list, got %d", listRR.Code)
	}
	list := decodeFileList(t, listRR)
	found := false
	for _, f := range list {
		if f.ID == meta.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("uploaded file %s not visible in GET /api/v1/files of owner, list: %+v", meta.ID, list)
	}
}

func TestUpload_IncrementsUsedBytes(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)
	content := []byte("quota accounting payload")

	before := getUserUsedBytes(t, pool, userID)
	rr := uploadTestFile(t, fh, userID, "quota.txt", content, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d, body: %s", rr.Code, rr.Body.String())
	}
	after := getUserUsedBytes(t, pool, userID)
	if after != before+int64(len(content)) {
		t.Errorf("expected used_bytes to grow from %d to %d, got %d", before, before+int64(len(content)), after)
	}
}

func TestUpload_SecondUploadOverRemainingQuota_413(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 1000) // total quota 1000 bytes

	userID := createTestUser(t, pool, 1000)

	// First upload consumes 400 bytes of the 1000-byte quota.
	first := bytes.Repeat([]byte("A"), 400)
	if rr := uploadTestFile(t, fh, userID, "first.bin", first, ""); rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first upload, got %d, body: %s", rr.Code, rr.Body.String())
	}
	usedAfterFirst := getUserUsedBytes(t, pool, userID)
	if usedAfterFirst != 400 {
		t.Fatalf("expected used_bytes 400 after first upload, got %d", usedAfterFirst)
	}

	// Second upload of 700 bytes fits the TOTAL quota (1000) but not the
	// REMAINING quota (600) — must be rejected with 413.
	second := bytes.Repeat([]byte("B"), 700)
	rr := uploadTestFile(t, fh, userID, "second.bin", second, "")
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 Payload Too Large for upload over remaining quota, got %d", rr.Code)
	}
	if got := getUserUsedBytes(t, pool, userID); got != usedAfterFirst {
		t.Errorf("expected used_bytes unchanged at %d after rejected upload, got %d", usedAfterFirst, got)
	}
	if got := countUserFiles(t, pool, userID); got != 1 {
		t.Errorf("expected exactly 1 files row after rejected upload, got %d", got)
	}
	if got := countRegularFiles(t, tempDir); got != 1 {
		t.Errorf("expected exactly 1 binary shard on disk after rejected upload, got %d", got)
	}
}

func TestUpload_ContentLengthPrecheckAgainstRemainingQuota(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10000) // total quota 10000 bytes

	userID := createTestUser(t, pool, 10000)

	// Consume 6000 bytes; remaining quota is 4000.
	if rr := uploadTestFile(t, fh, userID, "fill.bin", bytes.Repeat([]byte("F"), 6000), ""); rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for filler upload, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// Advertised Content-Length 5000 exceeds remaining 4000 but is BELOW the
	// total quota 10000 — the precheck must compare against the remainder and
	// reject before reading the body.
	req := buildUploadRequest(t, userID, "over.bin", bytes.Repeat([]byte("O"), 10), "")
	req.Header.Set("Content-Length", "5000")
	req.ContentLength = 5000
	rr := httptest.NewRecorder()
	fh.UploadHandler(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 from Content-Length precheck against remaining quota, got %d", rr.Code)
	}
	if got := getUserUsedBytes(t, pool, userID); got != 6000 {
		t.Errorf("expected used_bytes unchanged at 6000, got %d", got)
	}
	if got := countUserFiles(t, pool, userID); got != 1 {
		t.Errorf("expected exactly 1 files row, got %d", got)
	}
}

func TestUpload_MetadataPersistenceFailureRollsBackDisk(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)

	// Force files INSERT to fail at DB level AFTER the binary was written to disk
	// using a CHECK constraint, exercising the os.Remove + rollback + 500 path.
	_, err := pool.Exec(context.Background(), `ALTER TABLE files ADD CONSTRAINT test_force_meta_fail CHECK (filename != 'fail_meta.txt')`)
	if err != nil {
		t.Fatalf("failed to add constraint: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `ALTER TABLE files DROP CONSTRAINT IF EXISTS test_force_meta_fail`)
	})

	rr := uploadTestFile(t, fh, userID, "fail_meta.txt", []byte("written then rolled back"), "")

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when metadata persistence fails, got %d", rr.Code)
	}
	if got := getUserUsedBytes(t, pool, userID); got != 0 {
		t.Errorf("expected used_bytes unchanged at 0 after rollback, got %d", got)
	}
	if got := countUserFiles(t, pool, userID); got != 0 {
		t.Errorf("expected no files rows after rollback, got %d", got)
	}
	if got := countRegularFiles(t, tempDir); got != 0 {
		t.Errorf("expected written binary removed from disk after rollback, found %d file(s)", got)
	}
}

func TestUpload_EngineSaveErrorReturns500(t *testing.T) {
	pool := setupTestPool(t)
	unwritableEngine := storage.NewDiskEngine("/dev/null/invalid_path")
	fh := handler.NewFileHandler(unwritableEngine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)
	rr := uploadTestFile(t, fh, userID, "fail.txt", []byte("data"), "")

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when disk engine fails, got %d", rr.Code)
	}
	if got := getUserUsedBytes(t, pool, userID); got != 0 {
		t.Errorf("expected used_bytes unchanged at 0 after engine failure, got %d", got)
	}
}

func createMultipartRequestWithDiskTempFile(t *testing.T, userID uuid.UUID, filename string, content []byte, folderID string) (*http.Request, string, *multipart.FileHeader) {
	t.Helper()
	req := buildUploadRequest(t, userID, filename, content, folderID)
	// Parse with maxMemory = 0 to force storing file part in temporary disk file.
	if err := req.ParseMultipartForm(0); err != nil {
		t.Fatalf("failed to parse multipart form into temp file: %v", err)
	}
	if req.MultipartForm == nil || len(req.MultipartForm.File["file"]) == 0 {
		t.Fatalf("multipart form or file part is missing")
	}
	fh := req.MultipartForm.File["file"][0]
	f, err := fh.Open()
	if err != nil {
		t.Fatalf("failed to open multipart temp file: %v", err)
	}
	defer f.Close()

	osFile, ok := f.(*os.File)
	if !ok {
		t.Fatalf("expected *os.File for disk-backed multipart file header, got %T", f)
	}
	tempFilePath := osFile.Name()
	t.Cleanup(func() { _ = os.Remove(tempFilePath) })

	if _, err := os.Stat(tempFilePath); err != nil {
		t.Fatalf("expected temp file %s to exist before handler execution: %v", tempFilePath, err)
	}
	return req, tempFilePath, fh
}

func TestFileUpload_MultipartFormCleanup(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 50*1024*1024)

	t.Run("successful upload removes multipart temp files via RemoveAll", func(t *testing.T) {
		userID := createTestUser(t, pool, 10*1024*1024)
		req, tempPath, header := createMultipartRequestWithDiskTempFile(t, userID, "clean_success.txt", []byte("temp cleanup success payload"), "")

		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d, body: %s", rr.Code, rr.Body.String())
		}

		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Errorf("expected multipart temp file %s to be deleted by RemoveAll(), but stat err = %v", tempPath, err)
		}
		if f, err := header.Open(); err == nil {
			_ = f.Close()
			t.Errorf("expected header.Open() to fail after RemoveAll(), but succeeded")
		}
	})

	t.Run("validation error on invalid folder ID removes multipart temp files", func(t *testing.T) {
		userID := createTestUser(t, pool, 10*1024*1024)
		req, tempPath, header := createMultipartRequestWithDiskTempFile(t, userID, "invalid_folder.txt", []byte("temp cleanup validation error payload"), "not-a-valid-uuid")

		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
		}

		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Errorf("expected multipart temp file %s to be deleted on validation error, but stat err = %v", tempPath, err)
		}
		if f, err := header.Open(); err == nil {
			_ = f.Close()
			t.Errorf("expected header.Open() to fail after RemoveAll() on validation error, but succeeded")
		}
	})

	t.Run("folder not found error removes multipart temp files", func(t *testing.T) {
		userID := createTestUser(t, pool, 10*1024*1024)
		nonExistentFolderID := uuid.New().String()
		req, tempPath, header := createMultipartRequestWithDiskTempFile(t, userID, "missing_folder.txt", []byte("temp cleanup not found payload"), nonExistentFolderID)

		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", rr.Code)
		}

		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Errorf("expected multipart temp file %s to be deleted on 404 folder not found, but stat err = %v", tempPath, err)
		}
		if f, err := header.Open(); err == nil {
			_ = f.Close()
			t.Errorf("expected header.Open() to fail after RemoveAll() on 404 folder not found, but succeeded")
		}
	})

	t.Run("storage engine failure removes multipart temp files", func(t *testing.T) {
		userID := createTestUser(t, pool, 10*1024*1024)
		unwritableEngine := storage.NewDiskEngine("/dev/null/invalid_path")
		fhUnwritable := handler.NewFileHandler(unwritableEngine, pool, 10*1024*1024)
		req, tempPath, header := createMultipartRequestWithDiskTempFile(t, userID, "engine_fail.txt", []byte("temp cleanup engine error payload"), "")

		rr := httptest.NewRecorder()
		fhUnwritable.UploadHandler(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 Internal Server Error, got %d", rr.Code)
		}

		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Errorf("expected multipart temp file %s to be deleted on storage engine failure, but stat err = %v", tempPath, err)
		}
		if f, err := header.Open(); err == nil {
			_ = f.Close()
			t.Errorf("expected header.Open() to fail after RemoveAll() on storage failure, but succeeded")
		}
	})

	t.Run("large payload exceeding 32MB in-memory threshold cleans up temp file in TMPDIR", func(t *testing.T) {
		isolatedTmpDir := t.TempDir()
		t.Setenv("TMPDIR", isolatedTmpDir)

		userID := createTestUser(t, pool, 40*1024*1024)
		largePayload := bytes.Repeat([]byte("X"), 33*1024*1024)
		req := buildUploadRequest(t, userID, "large_33mb.bin", largePayload, "")

		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d, body: %s", rr.Code, rr.Body.String())
		}

		// Verify isolated TMPDIR does not leak multipart-* files
		entries, err := os.ReadDir(isolatedTmpDir)
		if err != nil {
			t.Fatalf("failed to read isolated TMPDIR: %v", err)
		}
		var leakedFiles []string
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "multipart-") {
				leakedFiles = append(leakedFiles, entry.Name())
			}
		}
		if len(leakedFiles) > 0 {
			t.Errorf("expected 0 leaked multipart temp files in TMPDIR after RemoveAll(), found %d: %v", len(leakedFiles), leakedFiles)
		}

		// Verify that if req.MultipartForm is still present on request, opening its file header fails
		if req.MultipartForm != nil {
			if fhs, ok := req.MultipartForm.File["file"]; ok && len(fhs) > 0 {
				if f, err := fhs[0].Open(); err == nil {
					_ = f.Close()
					t.Errorf("expected req.MultipartForm file header Open() to fail after RemoveAll(), but succeeded")
				}
			}
		}
	})
}

func TestFileUpload_IDOR_CrossUserFolderIsolation(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)
	folderB := createTestFolder(t, pool, userB)

	content := []byte("idor exploit payload")
	rr := uploadTestFile(t, fh, userA, "idor_attack.txt", content, folderB.String())
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found when User A uploads to User B's folder (IDOR), got %d, body: %s", rr.Code, rr.Body.String())
	}

	// Verify no file record was created in database for either user
	if got := countUserFiles(t, pool, userA); got != 0 {
		t.Errorf("expected 0 files for User A after rejected upload, got %d", got)
	}
	if got := countUserFiles(t, pool, userB); got != 0 {
		t.Errorf("expected 0 files for User B after rejected upload, got %d", got)
	}

	// Verify User A used_bytes quota was not charged
	if got := getUserUsedBytes(t, pool, userA); got != 0 {
		t.Errorf("expected used_bytes 0 for User A, got %d", got)
	}

	// Verify no binary shard was leaked to disk
	if got := countRegularFiles(t, tempDir); got != 0 {
		t.Errorf("expected no binary shards on disk after rejected IDOR upload, got %d", got)
	}
}
