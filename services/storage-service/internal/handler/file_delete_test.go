package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

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

	t.Run("deleting with non-UUID id returns 400 Bad Request", func(t *testing.T) {
		delRR := deleteFileRequest(fh, userA, "not-a-uuid")
		if delRR.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for non-UUID file ID, got %d", delRR.Code)
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
