package handler_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

// TestMultiTenancyFileIsolation verifies DB-backed file isolation between users:
// owner-only listing, IDOR download denial and unauthenticated upload rejection.
func TestMultiTenancyFileIsolation(t *testing.T) {
	pool := setupTestPool(t)
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, pool, 10*1024*1024)

	userA := createTestUser(t, pool, 10*1024*1024)
	userB := createTestUser(t, pool, 10*1024*1024)

	recUploadA := uploadTestFile(t, fileHandler, userA, "userA_secret.txt", []byte("User A secret payload"), "")
	if recUploadA.Code != http.StatusCreated {
		t.Fatalf("upload for User A failed, expected 201 Created, got %d, body: %s", recUploadA.Code, recUploadA.Body.String())
	}
	metaA := decodeFileMeta(t, recUploadA)

	t.Run("User B listing files does not see User A file", func(t *testing.T) {
		recListB := listFilesRequest(fileHandler, userB, "/api/v1/files")
		if recListB.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for User B list, got %d", recListB.Code)
		}
		filesB := decodeFileList(t, recListB)
		for _, f := range filesB {
			if f.ID == metaA.ID {
				t.Errorf("User B should NOT see User A's file %s in list", metaA.ID)
			}
		}
	})

	t.Run("User B downloading User A file returns 404 Not Found", func(t *testing.T) {
		recDlB := downloadFileRequest(fileHandler, userB, metaA.ID)
		if recDlB.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found when User B attempts to download User A file, got %d", recDlB.Code)
		}
	})

	t.Run("User B deleting User A file returns 404 Not Found", func(t *testing.T) {
		recDelB := deleteFileRequest(fileHandler, userB, metaA.ID)
		if recDelB.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found when User B attempts to delete User A file, got %d", recDelB.Code)
		}
	})

	t.Run("Unauthenticated request to upload fails with 401", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "anon.txt")
		_, _ = part.Write([]byte("anon data"))
		_ = writer.Close()

		req, err := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
		if err != nil {
			t.Fatalf("failed to create anonymous upload request: %v", err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		recUploadAnon := httptest.NewRecorder()
		fileHandler.UploadHandler(recUploadAnon, req)
		if recUploadAnon.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for request without user_id in context, got %d", recUploadAnon.Code)
		}
	})
}
