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
