package handler_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

func TestFileUploadHandler_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_handler_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, 10*1024*1024) // 10MB quota

	content := []byte("Sample file payload for upload testing.")
	hashBytes := sha256.Sum256(content)
	expectedSHA256 := hex.EncodeToString(hashBytes[:])

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "testfile.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(content)
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	if err != nil {
		t.Fatalf("failed to create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(fileHandler.UploadHandler)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 Created, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		ID       string `json:"id"`
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
		SHA256   string `json:"sha256"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to parse upload response JSON: %v", err)
	}

	if resp.Filename != "testfile.txt" {
		t.Errorf("expected filename 'testfile.txt', got %q", resp.Filename)
	}
	if resp.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), resp.Size)
	}
	if resp.SHA256 != expectedSHA256 {
		t.Errorf("expected SHA256 %s, got %s", expectedSHA256, resp.SHA256)
	}
}

func TestFileUploadHandler_QuotaExceeded(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_handler_quota_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, 50) // 50 bytes quota limit

	content := bytes.Repeat([]byte("X"), 500) // 500 bytes exceeds 50 byte quota

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "large.bin")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(content)
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(fileHandler.UploadHandler)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected HTTP 413 Payload Too Large, got %d", rr.Code)
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to parse JSON error payload: %v", err)
	}
	if errResp.Error == "" {
		t.Error("expected non-empty error message in response JSON")
	}
}

func TestFileDownloadHandler_SuccessAndNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_handler_dl_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, 1024*1024)

	fileID := "f47a8b90-1234-5678-9abc-def012345678"
	content := []byte("Downloaded binary content test")
	_, _, err = engine.Save(fileID, bytes.NewReader(content), 1024*1024)
	if err != nil {
		t.Fatalf("failed to setup test file: %v", err)
	}

	t.Run("Successful download", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/download/%s", fileID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		h := http.HandlerFunc(fileHandler.DownloadHandler)
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected HTTP 200 OK, got %d", rr.Code)
		}

		if rr.Header().Get("Content-Length") != fmt.Sprintf("%d", len(content)) {
			t.Errorf("expected Content-Length %d, got %s", len(content), rr.Header().Get("Content-Length"))
		}

		if rr.Body.String() != string(content) {
			t.Errorf("expected body %q, got %q", string(content), rr.Body.String())
		}
	})

	t.Run("File not found download", func(t *testing.T) {
		nonExistentID := "00000000-0000-0000-0000-000000000000"
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/download/%s", nonExistentID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		h := http.HandlerFunc(fileHandler.DownloadHandler)
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected HTTP 404 Not Found, got %d", rr.Code)
		}
	})
}

func TestFileListHandler_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_handler_list_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, 1024*1024)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(fileHandler.ListHandler)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d", rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var files []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&files); err != nil {
		t.Fatalf("failed to decode JSON list: %v", err)
	}
}
