package handler_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

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
