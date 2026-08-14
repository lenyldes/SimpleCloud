package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func TestFolderHandler_CreateSuccessRoot(t *testing.T) {
	folderHandler := handler.NewFolderHandler()
	testUserID := uuid.New()

	payload := map[string]interface{}{
		"name": "Documents",
	}
	jsonBody, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUserID(req.Context(), testUserID))

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(folderHandler.CreateHandler)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 Created, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp handler.FolderMetadata
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp.Name != "Documents" {
		t.Errorf("expected folder name 'Documents', got %q", resp.Name)
	}
	if resp.UserID != testUserID.String() {
		t.Errorf("expected user_id %s, got %s", testUserID.String(), resp.UserID)
	}
	if resp.ParentID != nil {
		t.Errorf("expected nil parent_id for root folder, got %v", resp.ParentID)
	}
}

func TestFolderHandler_CreateSuccessNested(t *testing.T) {
	folderHandler := handler.NewFolderHandler()
	testUserID := uuid.New()

	// First create parent folder
	rootPayload, _ := json.Marshal(map[string]interface{}{"name": "RootFolder"})
	reqRoot := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(rootPayload))
	reqRoot.Header.Set("Content-Type", "application/json")
	reqRoot = reqRoot.WithContext(auth.WithUserID(reqRoot.Context(), testUserID))
	rrRoot := httptest.NewRecorder()
	folderHandler.CreateHandler(rrRoot, reqRoot)

	var parentFolder handler.FolderMetadata
	_ = json.NewDecoder(rrRoot.Body).Decode(&parentFolder)

	// Now create nested subfolder
	subPayload, _ := json.Marshal(map[string]interface{}{
		"name":      "SubFolder",
		"parent_id": parentFolder.ID,
	})
	reqSub := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(subPayload))
	reqSub.Header.Set("Content-Type", "application/json")
	reqSub = reqSub.WithContext(auth.WithUserID(reqSub.Context(), testUserID))
	rrSub := httptest.NewRecorder()
	folderHandler.CreateHandler(rrSub, reqSub)

	if rrSub.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 Created for nested folder, got %d", rrSub.Code)
	}

	var childFolder handler.FolderMetadata
	if err := json.NewDecoder(rrSub.Body).Decode(&childFolder); err != nil {
		t.Fatalf("failed to decode child folder response: %v", err)
	}

	if childFolder.ParentID == nil || *childFolder.ParentID != parentFolder.ID {
		t.Errorf("expected parent_id %s, got %v", parentFolder.ID, childFolder.ParentID)
	}
}

func TestFolderHandler_CreateCrossUserParentForbidden(t *testing.T) {
	folderHandler := handler.NewFolderHandler()
	userA := uuid.New()
	userB := uuid.New()

	// User A creates folder
	payloadA, _ := json.Marshal(map[string]interface{}{"name": "UserA Folder"})
	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(payloadA))
	reqA.Header.Set("Content-Type", "application/json")
	reqA = reqA.WithContext(auth.WithUserID(reqA.Context(), userA))
	rrA := httptest.NewRecorder()
	folderHandler.CreateHandler(rrA, reqA)

	var folderA handler.FolderMetadata
	_ = json.NewDecoder(rrA.Body).Decode(&folderA)

	// User B attempts to create subfolder inside User A's folder
	payloadB, _ := json.Marshal(map[string]interface{}{
		"name":      "Hacker SubFolder",
		"parent_id": folderA.ID,
	})
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(payloadB))
	reqB.Header.Set("Content-Type", "application/json")
	reqB = reqB.WithContext(auth.WithUserID(reqB.Context(), userB))
	rrB := httptest.NewRecorder()
	folderHandler.CreateHandler(rrB, reqB)

	if rrB.Code != http.StatusNotFound && rrB.Code != http.StatusForbidden {
		t.Errorf("expected 404 or 403 when using another user's folder as parent_id, got %d", rrB.Code)
	}
}

func TestFolderHandler_CreateValidationErrors(t *testing.T) {
	folderHandler := handler.NewFolderHandler()
	testUserID := uuid.New()

	t.Run("Empty folder name returns 400", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{"name": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		folderHandler.CreateHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for empty name, got %d", rr.Code)
		}
	})

	t.Run("Unauthenticated request returns 401", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{"name": "Docs"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		folderHandler.CreateHandler(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for unauthenticated request, got %d", rr.Code)
		}
	})
}

func TestFolderHandler_ListScopedAndMultiTenant(t *testing.T) {
	folderHandler := handler.NewFolderHandler()
	userA := uuid.New()
	userB := uuid.New()

	// User A creates root folder and subfolder
	payloadA1, _ := json.Marshal(map[string]interface{}{"name": "RootA"})
	reqA1 := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(payloadA1))
	reqA1.Header.Set("Content-Type", "application/json")
	reqA1 = reqA1.WithContext(auth.WithUserID(reqA1.Context(), userA))
	rrA1 := httptest.NewRecorder()
	folderHandler.CreateHandler(rrA1, reqA1)

	var rootA handler.FolderMetadata
	_ = json.NewDecoder(rrA1.Body).Decode(&rootA)

	// List User A root folders
	reqListRoot := httptest.NewRequest(http.MethodGet, "/api/v1/folders", nil)
	reqListRoot = reqListRoot.WithContext(auth.WithUserID(reqListRoot.Context(), userA))
	rrListRoot := httptest.NewRecorder()
	folderHandler.ListHandler(rrListRoot, reqListRoot)

	if rrListRoot.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list root folders, got %d", rrListRoot.Code)
	}

	var rootFolders []handler.FolderMetadata
	_ = json.NewDecoder(rrListRoot.Body).Decode(&rootFolders)
	if len(rootFolders) != 1 || rootFolders[0].ID != rootA.ID {
		t.Errorf("expected 1 root folder %s, got %v", rootA.ID, rootFolders)
	}

	// User B list root folders (should see empty array)
	reqListB := httptest.NewRequest(http.MethodGet, "/api/v1/folders", nil)
	reqListB = reqListB.WithContext(auth.WithUserID(reqListB.Context(), userB))
	rrListB := httptest.NewRecorder()
	folderHandler.ListHandler(rrListB, reqListB)

	if rrListB.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for User B list, got %d", rrListB.Code)
	}

	var foldersB []handler.FolderMetadata
	_ = json.NewDecoder(rrListB.Body).Decode(&foldersB)
	if len(foldersB) != 0 {
		t.Errorf("User B should see 0 folders, got %d", len(foldersB))
	}
}

func TestFolderHandler_DeleteSuccessAndMultiTenant(t *testing.T) {
	folderHandler := handler.NewFolderHandler()
	userA := uuid.New()
	userB := uuid.New()

	payloadA, _ := json.Marshal(map[string]interface{}{"name": "FolderToDelete"})
	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(payloadA))
	reqA.Header.Set("Content-Type", "application/json")
	reqA = reqA.WithContext(auth.WithUserID(reqA.Context(), userA))
	rrA := httptest.NewRecorder()
	folderHandler.CreateHandler(rrA, reqA)

	var folderA handler.FolderMetadata
	_ = json.NewDecoder(rrA.Body).Decode(&folderA)

	// User B trying to delete User A folder returns 404
	reqDelB := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/"+folderA.ID, nil)
	reqDelB = reqDelB.WithContext(auth.WithUserID(reqDelB.Context(), userB))
	rrDelB := httptest.NewRecorder()
	folderHandler.DeleteHandler(rrDelB, reqDelB)

	if rrDelB.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found when User B deletes User A folder, got %d", rrDelB.Code)
	}

	// User A deletes their folder
	reqDelA := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/"+folderA.ID, nil)
	reqDelA = reqDelA.WithContext(auth.WithUserID(reqDelA.Context(), userA))
	rrDelA := httptest.NewRecorder()
	folderHandler.DeleteHandler(rrDelA, reqDelA)

	if rrDelA.Code != http.StatusOK {
		t.Errorf("expected 200 OK when User A deletes folder, got %d", rrDelA.Code)
	}
}

func TestFolderHandler_ConstructorAndEdgeCases(t *testing.T) {
	config, _ := pgxpool.ParseConfig("postgres://user:pass@127.0.0.1:1/dbname")
	pool, _ := pgxpool.NewWithConfig(context.Background(), config)
	if pool != nil {
		defer pool.Close()
	}

	tempDir, _ := os.MkdirTemp("", "folder_handler_test_*")
	defer os.RemoveAll(tempDir)
	engine := storage.NewDiskEngine(tempDir)

	fh := handler.NewFolderHandler(pool, engine)
	testUserID := uuid.New()

	t.Run("CreateHandler non-POST method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", rr.Code)
		}
	})

	t.Run("CreateHandler invalid JSON payload returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", strings.NewReader("{invalid_json"))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rr.Code)
		}
	})

	t.Run("CreateHandler non-existent parent_id returns 404", func(t *testing.T) {
		parentID := uuid.New().String()
		payload, _ := json.Marshal(map[string]interface{}{
			"name":      "NestedFolder",
			"parent_id": parentID,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fh.CreateHandler(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for non-existent parent_id, got %d", rr.Code)
		}
	})

	t.Run("ListHandler non-GET method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/folders", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", rr.Code)
		}
	})

	t.Run("ListHandler unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders", nil)
		rr := httptest.NewRecorder()
		fh.ListHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler non-DELETE method returns 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/folders/some-id", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler unauthenticated returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/some-id", nil)
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
		}
	})

	t.Run("DeleteHandler missing folder ID returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fh.DeleteHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for missing folder ID, got %d", rr.Code)
		}
	})
}

func TestFolderHandler_RecursiveSubfolderDeletion(t *testing.T) {
	fh := handler.NewFolderHandler()
	testUserID := uuid.New()

	// Create Folder Level 1
	p1, _ := json.Marshal(map[string]interface{}{"name": "Level1"})
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(p1))
	req1 = req1.WithContext(auth.WithUserID(req1.Context(), testUserID))
	rr1 := httptest.NewRecorder()
	fh.CreateHandler(rr1, req1)
	var f1 handler.FolderMetadata
	_ = json.NewDecoder(rr1.Body).Decode(&f1)

	// Create Folder Level 2 (parent = Level1)
	p2, _ := json.Marshal(map[string]interface{}{"name": "Level2", "parent_id": f1.ID})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(p2))
	req2 = req2.WithContext(auth.WithUserID(req2.Context(), testUserID))
	rr2 := httptest.NewRecorder()
	fh.CreateHandler(rr2, req2)
	var f2 handler.FolderMetadata
	_ = json.NewDecoder(rr2.Body).Decode(&f2)

	// Create Folder Level 3 (parent = Level2)
	p3, _ := json.Marshal(map[string]interface{}{"name": "Level3", "parent_id": f2.ID})
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/folders", bytes.NewReader(p3))
	req3 = req3.WithContext(auth.WithUserID(req3.Context(), testUserID))
	rr3 := httptest.NewRecorder()
	fh.CreateHandler(rr3, req3)
	var f3 handler.FolderMetadata
	_ = json.NewDecoder(rr3.Body).Decode(&f3)

	// Delete Level 1 -> should recursively delete Level 2 and Level 3
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/folders/"+f1.ID, nil)
	reqDel = reqDel.WithContext(auth.WithUserID(reqDel.Context(), testUserID))
	rrDel := httptest.NewRecorder()
	fh.DeleteHandler(rrDel, reqDel)

	if rrDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on recursive delete, got %d", rrDel.Code)
	}

	// Verify all folders deleted when listing Level 1's subfolders
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/folders?parent_id="+f1.ID, nil)
	reqList = reqList.WithContext(auth.WithUserID(reqList.Context(), testUserID))
	rrList := httptest.NewRecorder()
	fh.ListHandler(rrList, reqList)

	var subfolders []handler.FolderMetadata
	_ = json.NewDecoder(rrList.Body).Decode(&subfolders)
	if len(subfolders) != 0 {
		t.Errorf("expected 0 subfolders after recursive deletion, got %d", len(subfolders))
	}
}
