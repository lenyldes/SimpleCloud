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

	t.Run("ListHandler with all=true and nil pool returns empty list", func(t *testing.T) {
		fh := handler.NewFolderHandler(nil, engine)
		rr := listFoldersRequest(fh, userID, "/api/v1/folders?all=true")
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		list := decodeFolderList(t, rr)
		if len(list) != 0 {
			t.Errorf("expected empty list, got %d elements", len(list))
		}
	})

	t.Run("ListHandler with all=true and canceled context returns 500", func(t *testing.T) {
		pool := setupTestPool(t)
		fh := handler.NewFolderHandler(pool, engine)
		testUser := createTestUser(t, pool, 10*1024*1024)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders?all=true", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(auth.WithUserID(ctx, testUser))
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 on canceled query context with all=true, got %d", rr.Code)
		}
	})
}

// TestFolderList_AllParam_HierarchyAndTenantIsolation tests GET /api/v1/folders?all=true
// verifying that it returns all folders across all nesting levels for the authenticated
// user, strictly isolated from other users' folders.
func TestFolderList_AllParam_HierarchyAndTenantIsolation(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fh := handler.NewFolderHandler(pool, engine)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)

	// User A creates a 3-level folder hierarchy: RootA -> SubA1 -> SubA2
	rrRootA := createFolderRequest(fh, userA, map[string]interface{}{"name": "FolderA_Root"})
	if rrRootA.Code != http.StatusCreated {
		t.Fatalf("expected 201 for root folder A, got %d", rrRootA.Code)
	}
	rootA := decodeFolderMeta(t, rrRootA)

	rrChildA1 := createFolderRequest(fh, userA, map[string]interface{}{
		"name":      "FolderA_Sub1",
		"parent_id": rootA.ID,
	})
	if rrChildA1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for child folder A1, got %d", rrChildA1.Code)
	}
	childA1 := decodeFolderMeta(t, rrChildA1)

	rrChildA2 := createFolderRequest(fh, userA, map[string]interface{}{
		"name":      "FolderA_Sub2",
		"parent_id": childA1.ID,
	})
	if rrChildA2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for grandchild folder A2, got %d", rrChildA2.Code)
	}
	childA2 := decodeFolderMeta(t, rrChildA2)

	// User B creates a 2-level folder hierarchy: RootB -> SubB1
	rrRootB := createFolderRequest(fh, userB, map[string]interface{}{"name": "FolderB_Root"})
	if rrRootB.Code != http.StatusCreated {
		t.Fatalf("expected 201 for root folder B, got %d", rrRootB.Code)
	}
	rootB := decodeFolderMeta(t, rrRootB)

	rrChildB1 := createFolderRequest(fh, userB, map[string]interface{}{
		"name":      "FolderB_Sub1",
		"parent_id": rootB.ID,
	})
	if rrChildB1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for child folder B1, got %d", rrChildB1.Code)
	}
	childB1 := decodeFolderMeta(t, rrChildB1)

	t.Run("User A requesting all=true returns all 3 nested folders and zero User B folders", func(t *testing.T) {
		rr := listFoldersRequest(fh, userA, "/api/v1/folders?all=true")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}
		list := decodeFolderList(t, rr)
		if len(list) != 3 {
			t.Fatalf("expected all 3 folders for User A with ?all=true, got %d", len(list))
		}

		if list[0].ID != rootA.ID || list[0].Name != "FolderA_Root" || list[0].ParentID != nil {
			t.Errorf("expected root folder at index 0, got %+v", list[0])
		}
		if list[1].ID != childA1.ID || list[1].Name != "FolderA_Sub1" || list[1].ParentID == nil || *list[1].ParentID != rootA.ID {
			t.Errorf("expected child folder at index 1, got %+v", list[1])
		}
		if list[2].ID != childA2.ID || list[2].Name != "FolderA_Sub2" || list[2].ParentID == nil || *list[2].ParentID != childA1.ID {
			t.Errorf("expected grandchild folder at index 2, got %+v", list[2])
		}

		for _, f := range list {
			if f.UserID != userA.String() {
				t.Errorf("tenant leak: folder %s belongs to user %s, expected %s", f.ID, f.UserID, userA)
			}
			if f.ID == rootB.ID || f.ID == childB1.ID {
				t.Errorf("tenant leak: User B folder %s found in User A response", f.ID)
			}
		}
	})

	t.Run("User B requesting all=true returns all 2 nested folders and zero User A folders", func(t *testing.T) {
		rr := listFoldersRequest(fh, userB, "/api/v1/folders?all=true")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}
		list := decodeFolderList(t, rr)
		if len(list) != 2 {
			t.Fatalf("expected all 2 folders for User B with ?all=true, got %d", len(list))
		}
		for _, f := range list {
			if f.UserID != userB.String() {
				t.Errorf("tenant leak: folder %s belongs to user %s, expected %s", f.ID, f.UserID, userB)
			}
			if f.ID == rootA.ID || f.ID == childA1.ID || f.ID == childA2.ID {
				t.Errorf("tenant leak: User A folder %s found in User B response", f.ID)
			}
		}
	})

	t.Run("User A requesting without all=true returns only root folders (backward compatibility)", func(t *testing.T) {
		rr := listFoldersRequest(fh, userA, "/api/v1/folders")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}
		list := decodeFolderList(t, rr)
		if len(list) != 1 || list[0].ID != rootA.ID {
			t.Errorf("expected only root folder without ?all=true, got %+v", list)
		}
	})

	t.Run("User A requesting with all=false returns only root folders", func(t *testing.T) {
		rr := listFoldersRequest(fh, userA, "/api/v1/folders?all=false")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}
		list := decodeFolderList(t, rr)
		if len(list) != 1 || list[0].ID != rootA.ID {
			t.Errorf("expected only root folder for ?all=false, got %+v", list)
		}
	})

	t.Run("User A requesting with parent_id returns only direct child folders", func(t *testing.T) {
		rr := listFoldersRequest(fh, userA, "/api/v1/folders?parent_id="+rootA.ID)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}
		list := decodeFolderList(t, rr)
		if len(list) != 1 || list[0].ID != childA1.ID {
			t.Errorf("expected only direct child folder for parent_id query, got %+v", list)
		}
	})
}
