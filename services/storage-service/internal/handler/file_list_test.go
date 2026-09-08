package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

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
