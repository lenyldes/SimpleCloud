package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func TestMultiTenancyFileIsolation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multitenant_file_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, 10*1024*1024)

	userA := uuid.New()
	userB := uuid.New()

	// Upload file as User A
	bodyA := &bytes.Buffer{}
	writerA := multipart.NewWriter(bodyA)
	partA, _ := writerA.CreateFormFile("file", "userA_secret.txt")
	_, _ = partA.Write([]byte("User A secret payload"))
	_ = writerA.Close()

	reqUploadA := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", bodyA)
	reqUploadA.Header.Set("Content-Type", writerA.FormDataContentType())
	reqUploadA = reqUploadA.WithContext(auth.WithUserID(reqUploadA.Context(), userA))

	recUploadA := httptest.NewRecorder()
	fileHandler.UploadHandler(recUploadA, reqUploadA)

	if recUploadA.Code != http.StatusCreated {
		t.Fatalf("upload for User A failed, expected 201 Created, got %d", recUploadA.Code)
	}

	var metaA handler.FileMetadata
	if err := json.NewDecoder(recUploadA.Body).Decode(&metaA); err != nil {
		t.Fatalf("failed to parse User A upload response: %v", err)
	}

	t.Run("User B listing files does not see User A file", func(t *testing.T) {
		reqListB := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
		reqListB = reqListB.WithContext(auth.WithUserID(reqListB.Context(), userB))
		recListB := httptest.NewRecorder()

		fileHandler.ListHandler(recListB, reqListB)

		if recListB.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for User B list, got %d", recListB.Code)
		}

		var filesB []handler.FileMetadata
		if err := json.NewDecoder(recListB.Body).Decode(&filesB); err != nil {
			t.Fatalf("failed to decode User B file list: %v", err)
		}

		for _, f := range filesB {
			if f.ID == metaA.ID {
				t.Errorf("User B should NOT see User A's file %s in list", metaA.ID)
			}
		}
	})

	t.Run("User B downloading User A file returns 404 Not Found", func(t *testing.T) {
		reqDlB := httptest.NewRequest(http.MethodGet, "/api/v1/files/download/"+metaA.ID, nil)
		reqDlB = reqDlB.WithContext(auth.WithUserID(reqDlB.Context(), userB))
		recDlB := httptest.NewRecorder()

		fileHandler.DownloadHandler(recDlB, reqDlB)

		if recDlB.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found when User B attempts to download User A file, got %d", recDlB.Code)
		}
	})

	t.Run("Unauthenticated request to upload fails with 401", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "anon.txt")
		_, _ = part.Write([]byte("anon data"))
		_ = writer.Close()

		reqUploadAnon := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
		reqUploadAnon.Header.Set("Content-Type", writer.FormDataContentType())
		recUploadAnon := httptest.NewRecorder()

		fileHandler.UploadHandler(recUploadAnon, reqUploadAnon)

		if recUploadAnon.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for request without user_id in context, got %d", recUploadAnon.Code)
		}
	})
}
