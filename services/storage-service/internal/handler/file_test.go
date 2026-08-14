package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func decodeFileMeta(t *testing.T, rr *httptest.ResponseRecorder) handler.FileMetadata {
	t.Helper()
	var meta handler.FileMetadata
	if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode file metadata JSON: %v", err)
	}
	return meta
}

func decodeFileList(t *testing.T, rr *httptest.ResponseRecorder) []handler.FileMetadata {
	t.Helper()
	var list []handler.FileMetadata
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode file list JSON: %v", err)
	}
	return list
}

func listFilesRequest(fh *handler.FileHandler, userID uuid.UUID, url string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.ListHandler(rr, req)
	return rr
}

func downloadFileRequest(fh *handler.FileHandler, userID uuid.UUID, fileID string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodGet, "/api/v1/files/download/"+fileID, nil)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.DownloadHandler(rr, req)
	return rr
}

func deleteFileRequest(fh *handler.FileHandler, userID uuid.UUID, fileID string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodDelete, "/api/v1/files/"+fileID, nil)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.DeleteHandler(rr, req)
	return rr
}

// --- Non-DB guards: method and authentication checks only (pool may be nil) ---

func TestFileHandler_MethodAndAuthGuards(t *testing.T) {
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
	testUserID := uuid.New()

	withUser := func(req *http.Request) *http.Request {
		return req.WithContext(auth.WithUserID(req.Context(), testUserID))
	}

	t.Run("UploadHandler wrong method returns 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/upload", nil)
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, withUser(req))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler wrong method returns 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/download/1234", nil)
		rr := httptest.NewRecorder()
		fh.DownloadHandler(rr, withUser(req))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("ListHandler wrong method returns 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files", nil)
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, withUser(req))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler wrong method returns 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/"+uuid.New().String(), nil)
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, withUser(req))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload", strings.NewReader(""))
		rr := httptest.NewRecorder()
		fh.UploadHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/download/1234", nil)
		rr := httptest.NewRecorder()
		fh.DownloadHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("ListHandler unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/files/1234", nil)
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})
}

// --- Task 1.2: upload persists full metadata record into PostgreSQL ---

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

// --- Task 1.3: "restart" regression (C2) — fresh handler instance with same pool ---

func TestFileMetadata_SurvivesRestart(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)

	userID := createTestUser(t, pool, 10*1024*1024)
	content := []byte("restart survival content")

	first := handler.NewFileHandler(engine, pool, 10*1024*1024)
	rr := uploadTestFile(t, first, userID, "restart.txt", content, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d, body: %s", rr.Code, rr.Body.String())
	}
	meta := decodeFileMeta(t, rr)

	// Simulate service restart: brand-new handler instance sharing only the DB pool.
	second := handler.NewFileHandler(engine, pool, 10*1024*1024)

	listRR := listFilesRequest(second, userID, "/api/v1/files")
	list := decodeFileList(t, listRR)
	found := false
	for _, f := range list {
		if f.ID == meta.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("file %s not visible to owner after handler restart, list: %+v", meta.ID, list)
	}

	dlRR := downloadFileRequest(second, userID, meta.ID)
	if dlRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK download after restart, got %d", dlRR.Code)
	}
	if dlRR.Body.String() != string(content) {
		t.Errorf("expected downloaded body %q, got %q", content, dlRR.Body.String())
	}
}

// --- Task 1.4: IDOR regression (C2) ---

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

// --- Task 1.5: list returns only owner files with root/folder filters ---

func TestFileList_OwnerScopedAndFolderFilters(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)
	folderA := createTestFolder(t, pool, userA)

	if rr := uploadTestFile(t, fh, userA, "root-a.txt", []byte("root A"), ""); rr.Code != http.StatusCreated {
		t.Fatalf("root upload failed: %d", rr.Code)
	}
	if rr := uploadTestFile(t, fh, userA, "nested-a.txt", []byte("nested A"), folderA.String()); rr.Code != http.StatusCreated {
		t.Fatalf("nested upload failed: %d, body: %s", rr.Code, rr.Body.String())
	}
	if rr := uploadTestFile(t, fh, userB, "root-b.txt", []byte("root B"), ""); rr.Code != http.StatusCreated {
		t.Fatalf("user B upload failed: %d", rr.Code)
	}

	t.Run("root listing shows only owner's root files", func(t *testing.T) {
		list := decodeFileList(t, listFilesRequest(fh, userA, "/api/v1/files"))
		if len(list) != 1 || list[0].Filename != "root-a.txt" {
			t.Errorf("expected exactly [root-a.txt], got %+v", list)
		}
	})

	t.Run("empty folder_id filter means root", func(t *testing.T) {
		list := decodeFileList(t, listFilesRequest(fh, userA, "/api/v1/files?folder_id="))
		if len(list) != 1 || list[0].Filename != "root-a.txt" {
			t.Errorf("expected exactly [root-a.txt], got %+v", list)
		}
	})

	t.Run("folder_id filter shows only that folder's files", func(t *testing.T) {
		list := decodeFileList(t, listFilesRequest(fh, userA, "/api/v1/files?folder_id="+folderA.String()))
		if len(list) != 1 || list[0].Filename != "nested-a.txt" {
			t.Errorf("expected exactly [nested-a.txt], got %+v", list)
		}
	})

	t.Run("other user never sees foreign files", func(t *testing.T) {
		list := decodeFileList(t, listFilesRequest(fh, userB, "/api/v1/files"))
		for _, f := range list {
			if f.Filename == "root-a.txt" || f.Filename == "nested-a.txt" {
				t.Errorf("user B must not see user A file %q", f.Filename)
			}
		}
	})
}

// --- Tasks 2.1–2.4: transactional quota accounting (C4, M8) ---

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

	// folder_id referencing a non-existent folder forces the files INSERT to
	// fail (FK violation) AFTER the binary was written to disk — exercising
	// the os.Remove + rollback + 500 path required by the spec.
	missingFolder := uuid.New().String()
	rr := uploadTestFile(t, fh, userID, "doomed.txt", []byte("written then rolled back"), missingFolder)

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

// --- Tasks 3.1–3.3: DELETE /api/v1/files/:id (H4) ---

func TestFileDelete_OwnerSuccess(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)
	content := []byte("delete me")

	rr := uploadTestFile(t, fh, userID, "delete-me.txt", content, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rr.Code)
	}
	meta := decodeFileMeta(t, rr)

	if _, err := engine.GetFilePath(meta.ID); err != nil {
		t.Fatalf("expected binary shard on disk before delete: %v", err)
	}

	delRR := deleteFileRequest(fh, userID, meta.ID)
	if delRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for owner delete, got %d, body: %s", delRR.Code, delRR.Body.String())
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM files WHERE id = $1`, meta.ID).Scan(&count); err != nil {
		t.Fatalf("failed to query files row: %v", err)
	}
	if count != 0 {
		t.Errorf("expected files row deleted, found %d rows", count)
	}

	if _, err := engine.GetFilePath(meta.ID); err == nil {
		t.Error("expected binary shard removed from disk after delete")
	}

	if got := getUserUsedBytes(t, pool, userID); got != 0 {
		t.Errorf("expected used_bytes decremented to 0, got %d", got)
	}
}

func TestFileDelete_ForeignAndInvalidIDs(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)

	rr := uploadTestFile(t, fh, userA, "keep.txt", []byte("must survive"), "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rr.Code)
	}
	meta := decodeFileMeta(t, rr)

	t.Run("deleting another user's file returns 404 and touches nothing", func(t *testing.T) {
		delRR := deleteFileRequest(fh, userB, meta.ID)
		if delRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 for foreign file delete, got %d", delRR.Code)
		}
		if got := countUserFiles(t, pool, userA); got != 1 {
			t.Errorf("expected files row untouched, got %d rows", got)
		}
		if _, err := engine.GetFilePath(meta.ID); err != nil {
			t.Errorf("expected binary shard untouched: %v", err)
		}
		if got := getUserUsedBytes(t, pool, userA); got != int64(len("must survive")) {
			t.Errorf("expected used_bytes untouched, got %d", got)
		}
	})

	t.Run("deleting non-existent UUID returns 404", func(t *testing.T) {
		delRR := deleteFileRequest(fh, userA, uuid.New().String())
		if delRR.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", delRR.Code)
		}
	})

	t.Run("deleting with non-UUID id returns 404", func(t *testing.T) {
		delRR := deleteFileRequest(fh, userA, "not-a-uuid")
		if delRR.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", delRR.Code)
		}
	})
}

func TestFileDelete_UsedBytesNeverNegative(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)

	// Insert an inconsistent files row whose size exceeds used_bytes (0).
	fileID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO files (id, user_id, filename, size_bytes, sha256_hash, storage_path)
		 VALUES ($1, $2, 'inconsistent.bin', 500, repeat('0', 64), '/nonexistent/path')`,
		fileID, userID)
	if err != nil {
		t.Fatalf("failed to insert inconsistent files row: %v", err)
	}

	delRR := deleteFileRequest(fh, userID, fileID.String())
	if delRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", delRR.Code, delRR.Body.String())
	}

	if got := getUserUsedBytes(t, pool, userID); got != 0 {
		t.Errorf("expected used_bytes clamped at 0, got %d", got)
	}
}

// --- Task 3.6: DELETE route must not conflict with download/ and list routes ---

func TestFilesRouting_NoConflict(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)

	// Wire the mux exactly as cmd/main.go is required to: the more specific
	// /api/v1/files/download/ pattern wins over the /api/v1/files/ dispatcher,
	// DELETE on a single path segment reaches DeleteHandler.
	mux := http.NewServeMux()
	mux.Handle("/api/v1/files/upload", http.HandlerFunc(fh.UploadHandler))
	mux.Handle("/api/v1/files/download/", http.HandlerFunc(fh.DownloadHandler))
	mux.Handle("/api/v1/files", http.HandlerFunc(fh.ListHandler))
	mux.Handle("/api/v1/files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segment := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/files/"), "/")
		if r.Method == http.MethodDelete && segment != "" && !strings.Contains(segment, "/") {
			fh.DeleteHandler(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))
	content := []byte("routing test payload")
	rr := uploadTestFile(t, fh, userID, "routing.txt", content, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rr.Code)
	}
	meta := decodeFileMeta(t, rr)

	doRequest := func(method, path string, fileID string) *http.Response {
		req := httptest.NewRequest(method, path+fileID, nil)
		req = req.WithContext(auth.WithUserID(req.Context(), userID))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Result()
	}

	// GET download route still served by DownloadHandler, not the dispatcher.
	resp := doRequest(http.MethodGet, "/api/v1/files/download/", meta.ID)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from download route, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// GET list route unaffected.
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
	reqList = reqList.WithContext(auth.WithUserID(reqList.Context(), userID))
	recList := httptest.NewRecorder()
	mux.ServeHTTP(recList, reqList)
	respList := recList.Result()
	if respList.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from list route, got %d", respList.StatusCode)
	}
	respList.Body.Close()

	// DELETE single segment reaches DeleteHandler.
	respDel := doRequest(http.MethodDelete, "/api/v1/files/", meta.ID)
	if respDel.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from DELETE /api/v1/files/:id, got %d", respDel.StatusCode)
	}
	respDel.Body.Close()

	if got := countUserFiles(t, pool, userID); got != 0 {
		t.Errorf("expected file deleted via routed DELETE, got %d rows", got)
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

	t.Run("UploadHandler non-UUID string folder_id triggers FK/exec error -> 500 and disk rollback", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		rr := uploadTestFile(t, fh, testUser, "nonuuid_folder.txt", []byte("content"), "invalid-folder-uuid")
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 for non-UUID string folder_id causing DB insert failure, got %d", rr.Code)
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

func TestFileHandler_ListHandler_BranchCoverage(t *testing.T) {
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	userID := uuid.New()

	t.Run("ListHandler pool is nil returns empty list", func(t *testing.T) {
		fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
		rr := listFilesRequest(fh, userID, "/api/v1/files")
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		list := decodeFileList(t, rr)
		if len(list) != 0 {
			t.Errorf("expected empty list, got %d elements", len(list))
		}
	})

	t.Run("ListHandler non-UUID folder_id returns empty list", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		rr := listFilesRequest(fh, testUser, "/api/v1/files?folder_id=not-a-uuid")
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		list := decodeFileList(t, rr)
		if len(list) != 0 {
			t.Errorf("expected empty list for invalid folder_id UUID, got %d elements", len(list))
		}
	})

	t.Run("ListHandler canceled context on query returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(auth.WithUserID(ctx, testUser))
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 on canceled query context, got %d", rr.Code)
		}
	})
}

func TestFileHandler_DeleteHandler_BranchCoverage(t *testing.T) {
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	userID := uuid.New()

	t.Run("DeleteHandler empty file ID returns 400", func(t *testing.T) {
		fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
		rr := deleteFileRequest(fh, userID, "")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty file ID, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler pool is nil returns 404", func(t *testing.T) {
		fh := handler.NewFileHandler(engine, nil, 10*1024*1024)
		rr := deleteFileRequest(fh, userID, uuid.New().String())
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 when pool is nil, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler binary missing on disk logs warning and returns 200", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		fileID := uuid.New()
		missingStoragePath := "/nonexistent/path/to/already_deleted.bin"
		_, err := pool.Exec(context.Background(),
			`INSERT INTO files (id, user_id, filename, size_bytes, sha256_hash, storage_path)
			 VALUES ($1, $2, 'already_deleted.bin', 100, repeat('b', 64), $3)`,
			fileID, testUser, missingStoragePath)
		if err != nil {
			t.Fatalf("failed to insert missing file row: %v", err)
		}

		rr := deleteFileRequest(fh, testUser, fileID.String())
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 OK when binary on disk is missing during delete, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler canceled context on tx begin returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFileHandler(engine, pool, 10*1024*1024)
		testUser := createTestUser(t, pool, 10*1024*1024)
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/files/"+uuid.New().String(), nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(auth.WithUserID(ctx, testUser))
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, req)
		if rr.Code != http.StatusNotFound && rr.Code != http.StatusInternalServerError {
			t.Errorf("expected error code for canceled context delete, got %d", rr.Code)
		}
	})
}
