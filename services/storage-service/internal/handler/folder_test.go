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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func createFolderRequest(fh *handler.FolderHandler, userID uuid.UUID, payload map[string]interface{}) *httptest.ResponseRecorder {
	jsonBody, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.CreateHandler(rr, req)
	return rr
}

func listFoldersRequest(fh *handler.FolderHandler, userID uuid.UUID, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.ListHandler(rr, req)
	return rr
}

func deleteFolderRequest(fh *handler.FolderHandler, userID uuid.UUID, folderID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/"+folderID, nil)
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.DeleteHandler(rr, req)
	return rr
}

func decodeFolderMeta(t *testing.T, rr *httptest.ResponseRecorder) handler.FolderMetadata {
	t.Helper()
	var meta handler.FolderMetadata
	if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode folder metadata JSON: %v", err)
	}
	return meta
}

func decodeFolderList(t *testing.T, rr *httptest.ResponseRecorder) []handler.FolderMetadata {
	t.Helper()
	var list []handler.FolderMetadata
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode folder list JSON: %v", err)
	}
	return list
}

func folderExistsInDB(t *testing.T, pool *pgxpool.Pool, folderID string) bool {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM folders WHERE id = $1`, folderID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query folders row: %v", err)
	}
	return count > 0
}

// --- Non-DB guards: method, auth and payload validation (pool may be nil) ---

func TestFolderHandler_MethodAndAuthGuards(t *testing.T) {
	fh := handler.NewFolderHandler(nil, nil)
	testUserID := uuid.New()

	withUser := func(req *http.Request) *http.Request {
		return req.WithContext(auth.WithUserID(req.Context(), testUserID))
	}

	t.Run("CreateHandler wrong method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders", nil)
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, withUser(req))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("ListHandler wrong method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", nil)
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, withUser(req))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler wrong method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders/some-id", nil)
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, withUser(req))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("CreateHandler unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", strings.NewReader("{}"))
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("ListHandler unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders", nil)
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/some-id", nil)
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("CreateHandler invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", strings.NewReader("{invalid_json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, withUser(req))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("CreateHandler empty name returns 400", func(t *testing.T) {
		rr := createFolderRequest(fh, testUserID, map[string]interface{}{"name": "  "})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}

// --- Task 4.1: non-UUID parent_id rejected with 400 (M7) ---

func TestFolderCreate_InvalidParentUUID400(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFolderHandler(pool, engine)

	userID := createTestUser(t, pool, 10*1024*1024)

	rr := createFolderRequest(fh, userID, map[string]interface{}{
		"name":      "BrokenParent",
		"parent_id": "not-a-uuid",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-UUID parent_id, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if !strings.Contains(errResp.Error, "invalid parent_id") {
		t.Errorf("expected error to mention 'invalid parent_id', got %q", errResp.Error)
	}

	if got := countUserFolders(t, pool, userID); got != 0 {
		t.Errorf("expected no folder record created for invalid parent_id, found %d", got)
	}
}

// --- Task 4.2: parent existence/ownership verified in PostgreSQL ---

func TestFolderCreate_ParentChecksViaDB(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFolderHandler(pool, engine)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)

	rrRoot := createFolderRequest(fh, userA, map[string]interface{}{"name": "RootA"})
	if rrRoot.Code != http.StatusCreated {
		t.Fatalf("expected 201 for root folder, got %d, body: %s", rrRoot.Code, rrRoot.Body.String())
	}
	rootA := decodeFolderMeta(t, rrRoot)
	if rootA.ParentID != nil {
		t.Errorf("expected nil parent_id for root folder, got %v", rootA.ParentID)
	}
	if !folderExistsInDB(t, pool, rootA.ID) {
		t.Errorf("expected folder %s persisted in folders table", rootA.ID)
	}

	t.Run("own parent exists -> 201 nested", func(t *testing.T) {
		rr := createFolderRequest(fh, userA, map[string]interface{}{
			"name":      "ChildA",
			"parent_id": rootA.ID,
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 for nested folder under own parent, got %d, body: %s", rr.Code, rr.Body.String())
		}
		child := decodeFolderMeta(t, rr)
		if child.ParentID == nil || *child.ParentID != rootA.ID {
			t.Errorf("expected parent_id %s, got %v", rootA.ID, child.ParentID)
		}
	})

	t.Run("foreign parent -> 404", func(t *testing.T) {
		rr := createFolderRequest(fh, userB, map[string]interface{}{
			"name":      "HackerChild",
			"parent_id": rootA.ID,
		})
		if rr.Code != http.StatusNotFound && rr.Code != http.StatusForbidden {
			t.Errorf("expected 404/403 for foreign parent, got %d", rr.Code)
		}
		if got := countUserFolders(t, pool, userB); got != 0 {
			t.Errorf("expected no folder created for foreign parent, found %d", got)
		}
	})

	t.Run("non-existent parent UUID -> 404", func(t *testing.T) {
		rr := createFolderRequest(fh, userA, map[string]interface{}{
			"name":      "OrphanChild",
			"parent_id": uuid.New().String(),
		})
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for non-existent parent, got %d", rr.Code)
		}
	})
}

// --- Task 4.3: folder metadata survives handler restart (C2) ---

func TestFolderMetadata_SurvivesRestart(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)

	userID := createTestUser(t, pool, 10*1024*1024)

	first := handler.NewFolderHandler(pool, engine)
	rrRoot := createFolderRequest(first, userID, map[string]interface{}{"name": "PersistRoot"})
	if rrRoot.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", rrRoot.Code, rrRoot.Body.String())
	}
	root := decodeFolderMeta(t, rrRoot)

	rrChild := createFolderRequest(first, userID, map[string]interface{}{
		"name":      "PersistChild",
		"parent_id": root.ID,
	})
	if rrChild.Code != http.StatusCreated {
		t.Fatalf("expected 201 for child, got %d", rrChild.Code)
	}
	child := decodeFolderMeta(t, rrChild)

	// Simulate service restart: fresh handler instance sharing only the pool.
	second := handler.NewFolderHandler(pool, engine)

	rootList := decodeFolderList(t, listFoldersRequest(second, userID, "/api/v1/folders"))
	if len(rootList) != 1 || rootList[0].ID != root.ID {
		t.Errorf("expected exactly root folder %s after restart, got %+v", root.ID, rootList)
	}

	childList := decodeFolderList(t, listFoldersRequest(second, userID, "/api/v1/folders?parent_id="+root.ID))
	if len(childList) != 1 || childList[0].ID != child.ID {
		t.Errorf("expected exactly child folder %s after restart, got %+v", child.ID, childList)
	}
}

// --- Task 3.4: recursive folder deletion with disk and quota cleanup (H4) ---

func TestFolderDelete_RecursiveWithDiskCleanup(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	folderHandler := handler.NewFolderHandler(pool, engine)
	fileHandler := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)

	rrL1 := createFolderRequest(folderHandler, userID, map[string]interface{}{"name": "Level1"})
	if rrL1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for Level1, got %d", rrL1.Code)
	}
	l1 := decodeFolderMeta(t, rrL1)

	rrL2 := createFolderRequest(folderHandler, userID, map[string]interface{}{"name": "Level2", "parent_id": l1.ID})
	if rrL2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for Level2, got %d, body: %s", rrL2.Code, rrL2.Body.String())
	}
	l2 := decodeFolderMeta(t, rrL2)

	content := []byte("nested binary payload")
	rrFile := uploadTestFile(t, fileHandler, userID, "nested.bin", content, l2.ID)
	if rrFile.Code != http.StatusCreated {
		t.Fatalf("expected 201 for nested file upload, got %d, body: %s", rrFile.Code, rrFile.Body.String())
	}
	fileMeta := decodeFileMeta(t, rrFile)

	if got := getUserUsedBytes(t, pool, userID); got != int64(len(content)) {
		t.Fatalf("expected used_bytes %d before delete, got %d", len(content), got)
	}
	if _, err := engine.GetFilePath(fileMeta.ID); err != nil {
		t.Fatalf("expected nested file shard on disk before delete: %v", err)
	}

	delRR := deleteFolderRequest(folderHandler, userID, l1.ID)
	if delRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for recursive delete, got %d, body: %s", delRR.Code, delRR.Body.String())
	}

	if folderExistsInDB(t, pool, l1.ID) {
		t.Error("expected Level1 folder row deleted")
	}
	if folderExistsInDB(t, pool, l2.ID) {
		t.Error("expected nested Level2 folder row deleted recursively")
	}
	if got := countUserFiles(t, pool, userID); got != 0 {
		t.Errorf("expected nested files rows deleted, found %d", got)
	}
	if _, err := engine.GetFilePath(fileMeta.ID); err == nil {
		t.Error("expected nested file binary shard removed from disk")
	}
	if got := countRegularFiles(t, tempDir); got != 0 {
		t.Errorf("expected no binaries left on disk after recursive delete, found %d", got)
	}
	if got := getUserUsedBytes(t, pool, userID); got != 0 {
		t.Errorf("expected used_bytes adjusted to 0 after recursive delete, got %d", got)
	}
}

// --- Task 3.5: foreign folder delete denied; deleted files no longer downloadable ---

func TestFolderDelete_ForeignDeniedAndFilesGone(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	folderHandler := handler.NewFolderHandler(pool, engine)
	fileHandler := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)

	rrFolder := createFolderRequest(folderHandler, userA, map[string]interface{}{"name": "A Folder"})
	if rrFolder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rrFolder.Code)
	}
	folderA := decodeFolderMeta(t, rrFolder)

	content := []byte("A's nested file")
	rrFile := uploadTestFile(t, fileHandler, userA, "inside.bin", content, folderA.ID)
	if rrFile.Code != http.StatusCreated {
		t.Fatalf("expected 201 for file upload, got %d, body: %s", rrFile.Code, rrFile.Body.String())
	}
	fileA := decodeFileMeta(t, rrFile)

	t.Run("foreign folder delete returns 404 and leaves everything intact", func(t *testing.T) {
		delRR := deleteFolderRequest(folderHandler, userB, folderA.ID)
		if delRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 for foreign folder delete, got %d", delRR.Code)
		}
		if !folderExistsInDB(t, pool, folderA.ID) {
			t.Error("expected folder row untouched after foreign delete attempt")
		}
		if got := countUserFiles(t, pool, userA); got != 1 {
			t.Errorf("expected file row untouched, got %d", got)
		}
		if _, err := engine.GetFilePath(fileA.ID); err != nil {
			t.Errorf("expected binary shard untouched: %v", err)
		}
	})

	t.Run("unknown folder id returns 404", func(t *testing.T) {
		delRR := deleteFolderRequest(folderHandler, userA, uuid.New().String())
		if delRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 for unknown folder, got %d", delRR.Code)
		}
	})

	t.Run("owner delete succeeds and files of deleted folder are not downloadable", func(t *testing.T) {
		delRR := deleteFolderRequest(folderHandler, userA, folderA.ID)
		if delRR.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d, body: %s", delRR.Code, delRR.Body.String())
		}

		dlRR := downloadFileRequest(fileHandler, userA, fileA.ID)
		if dlRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 downloading file of deleted folder, got %d", dlRR.Code)
		}
		if _, err := engine.GetFilePath(fileA.ID); err == nil {
			t.Error("expected binary shard of deleted folder removed from disk")
		}
	})
}

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

func TestFolderDelete_MultiLevelRecursion(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	folderHandler := handler.NewFolderHandler(pool, engine)
	fileHandler := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userID := createTestUser(t, pool, 10*1024*1024)

	rrL1 := createFolderRequest(folderHandler, userID, map[string]interface{}{"name": "TreeL1"})
	if rrL1.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rrL1.Code)
	}
	l1 := decodeFolderMeta(t, rrL1)

	rrL2a := createFolderRequest(folderHandler, userID, map[string]interface{}{"name": "TreeL2a", "parent_id": l1.ID})
	if rrL2a.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rrL2a.Code)
	}
	l2a := decodeFolderMeta(t, rrL2a)

	rrL2b := createFolderRequest(folderHandler, userID, map[string]interface{}{"name": "TreeL2b", "parent_id": l1.ID})
	if rrL2b.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rrL2b.Code)
	}
	l2b := decodeFolderMeta(t, rrL2b)

	rrL3 := createFolderRequest(folderHandler, userID, map[string]interface{}{"name": "TreeL3", "parent_id": l2a.ID})
	if rrL3.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rrL3.Code)
	}
	l3 := decodeFolderMeta(t, rrL3)

	_ = uploadTestFile(t, fileHandler, userID, "f2b.txt", []byte("file in l2b"), l2b.ID)
	_ = uploadTestFile(t, fileHandler, userID, "f3.txt", []byte("file in l3"), l3.ID)

	delRR := deleteFolderRequest(folderHandler, userID, l1.ID)
	if delRR.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for deep multi-level recursive delete, got %d, body: %s", delRR.Code, delRR.Body.String())
	}

	if folderExistsInDB(t, pool, l1.ID) || folderExistsInDB(t, pool, l2a.ID) || folderExistsInDB(t, pool, l2b.ID) || folderExistsInDB(t, pool, l3.ID) {
		t.Error("expected all multi-level folders deleted")
	}
	if got := countUserFiles(t, pool, userID); got != 0 {
		t.Errorf("expected 0 files remaining, got %d", got)
	}
	if got := getUserUsedBytes(t, pool, userID); got != 0 {
		t.Errorf("expected used_bytes 0 after multi-level delete, got %d", got)
	}
}
