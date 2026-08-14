package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
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
