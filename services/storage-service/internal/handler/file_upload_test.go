package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	// Filename exceeding VARCHAR(512) forces the files INSERT to fail at DB
	// level AFTER the binary was written to disk — exercising the
	// os.Remove + rollback + 500 path required by the spec.
	longFilename := strings.Repeat("a", 600) + ".txt"
	rr := uploadTestFile(t, fh, userID, longFilename, []byte("written then rolled back"), "")

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
}
