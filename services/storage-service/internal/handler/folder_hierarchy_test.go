package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func TestFolderHandler_BranchCoverage(t *testing.T) {
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	userID := uuid.New()

	t.Run("CreateHandler with nil pool succeeds creating metadata", func(t *testing.T) {
		fh := handler.NewFolderHandler(nil, engine)
		rr := createFolderRequest(fh, userID, map[string]interface{}{"name": "PoolNilFolder"})
		if rr.Code != http.StatusCreated {
			t.Errorf("expected 201 when pool is nil, got %d", rr.Code)
		}
	})

	t.Run("CreateHandler canceled context on pool exec returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFolderHandler(pool, engine)
		testUser := createTestUser(t, pool, 10*1024*1024)
		jsonBody, _ := json.Marshal(map[string]interface{}{"name": "CanceledFolder"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(auth.WithUserID(ctx, testUser))
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 when folder insert fails on canceled context, got %d", rr.Code)
		}
	})

	t.Run("ListHandler pool is nil returns empty list", func(t *testing.T) {
		fh := handler.NewFolderHandler(nil, engine)
		rr := listFoldersRequest(fh, userID, "/api/v1/folders")
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		list := decodeFolderList(t, rr)
		if len(list) != 0 {
			t.Errorf("expected empty list, got %d elements", len(list))
		}
	})

	t.Run("ListHandler non-UUID parent_id returns empty list", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFolderHandler(pool, engine)
		testUser := createTestUser(t, pool, 10*1024*1024)
		rr := listFoldersRequest(fh, testUser, "/api/v1/folders?parent_id=invalid-uuid")
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		list := decodeFolderList(t, rr)
		if len(list) != 0 {
			t.Errorf("expected empty list for non-UUID parent_id, got %d elements", len(list))
		}
	})

	t.Run("ListHandler canceled context on query returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFolderHandler(pool, engine)
		testUser := createTestUser(t, pool, 10*1024*1024)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(auth.WithUserID(ctx, testUser))
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 on canceled query context, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler empty folder ID returns 400", func(t *testing.T) {
		fh := handler.NewFolderHandler(nil, engine)
		rr := deleteFolderRequest(fh, userID, "")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty folder ID, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler non-UUID folder ID returns 400 Bad Request", func(t *testing.T) {
		fh := handler.NewFolderHandler(nil, engine)
		rr := deleteFolderRequest(fh, userID, "not-a-uuid")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for non-UUID folder ID, got %d", rr.Code)
		}
	})

	t.Run("CreateHandler payload larger than 1MB returns 400 or 413", func(t *testing.T) {
		fh := handler.NewFolderHandler(nil, engine)
		hugeName := strings.Repeat("A", 1024*1024+100)
		jsonBody, _ := json.Marshal(map[string]interface{}{"name": hugeName})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), userID))
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, req)
		if rr.Code != http.StatusBadRequest && rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected 400 or 413 for payload > 1MB, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler pool is nil returns 404", func(t *testing.T) {
		fh := handler.NewFolderHandler(nil, engine)
		rr := deleteFolderRequest(fh, userID, uuid.New().String())
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 when pool is nil, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler subfolder binary missing on disk logs warning and returns 200", func(t *testing.T) {
		pool := setupTestPool(t)
		folderFH := handler.NewFolderHandler(pool, engine)
		testUser := createTestUser(t, pool, 10*1024*1024)

		rrFolder := createFolderRequest(folderFH, testUser, map[string]interface{}{"name": "MissingDiskFolder"})
		if rrFolder.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rrFolder.Code)
		}
		folderMeta := decodeFolderMeta(t, rrFolder)

		fileID := uuid.New()
		missingStoragePath := "/nonexistent/path/to/subfolder_file.bin"
		_, err := pool.Exec(context.Background(),
			`INSERT INTO files (id, user_id, folder_id, filename, size_bytes, sha256_hash, storage_path)
			 VALUES ($1, $2, $3, 'subfile.bin', 50, repeat('c', 64), $4)`,
			fileID, testUser, folderMeta.ID, missingStoragePath)
		if err != nil {
			t.Fatalf("failed to insert subfile row: %v", err)
		}
		_, _ = pool.Exec(context.Background(), `UPDATE users SET used_bytes = 50 WHERE id = $1`, testUser)

		delRR := deleteFolderRequest(folderFH, testUser, folderMeta.ID)
		if delRR.Code != http.StatusOK {
			t.Errorf("expected 200 OK when disk binary of deleted subfolder is missing, got %d, body: %s", delRR.Code, delRR.Body.String())
		}
		if got := getUserUsedBytes(t, pool, testUser); got != 0 {
			t.Errorf("expected used_bytes decremented to 0, got %d", got)
		}
	})

	t.Run("DeleteHandler canceled context on tx begin returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFolderHandler(pool, engine)
		testUser := createTestUser(t, pool, 10*1024*1024)

		rrFolder := createFolderRequest(fh, testUser, map[string]interface{}{"name": "TargetFolder"})
		if rrFolder.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rrFolder.Code)
		}
		folderMeta := decodeFolderMeta(t, rrFolder)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/"+folderMeta.ID, nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(auth.WithUserID(ctx, testUser))
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, req)
		if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusNotFound {
			t.Errorf("expected error code on canceled context delete, got %d", rr.Code)
		}
	})
}
