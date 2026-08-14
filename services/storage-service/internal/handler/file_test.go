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
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
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
	req = req.WithContext(auth.WithUserID(req.Context(), uuid.New()))

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
	req = req.WithContext(auth.WithUserID(req.Context(), uuid.New()))

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

	testUserID := uuid.New()

	t.Run("Successful download", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/download/%s", fileID), nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))

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
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))

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
	req = req.WithContext(auth.WithUserID(req.Context(), uuid.New()))

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

func TestFileHandler_InvalidMethodsAndInputs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_handler_err_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, 1024*1024)
	testUserID := uuid.New()

	t.Run("UploadHandler method not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/upload", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fileHandler.UploadHandler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler invalid multipart form", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload", strings.NewReader("not a multipart body"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fileHandler.UploadHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("UploadHandler missing file field", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("other", "value")
		_ = writer.Close()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fileHandler.UploadHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler method not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/download/1234", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fileHandler.DownloadHandler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler missing file ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/download/", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fileHandler.DownloadHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("ListHandler method not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/files", nil)
		req = req.WithContext(auth.WithUserID(req.Context(), testUserID))
		rr := httptest.NewRecorder()
		fileHandler.ListHandler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files/download/1234", nil)
		rr := httptest.NewRecorder()
		fileHandler.DownloadHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("ListHandler unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
		rr := httptest.NewRecorder()
		fileHandler.ListHandler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("DownloadHandler another user file returns 404", func(t *testing.T) {
		userA := uuid.New()
		userB := uuid.New()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "usera.txt")
		_, _ = part.Write([]byte("data"))
		_ = writer.Close()

		uploadReq, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
		uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
		uploadReq = uploadReq.WithContext(auth.WithUserID(uploadReq.Context(), userA))
		uploadRR := httptest.NewRecorder()
		fileHandler.UploadHandler(uploadRR, uploadReq)

		var meta handler.FileMetadata
		_ = json.NewDecoder(uploadRR.Body).Decode(&meta)

		dlReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/download/%s", meta.ID), nil)
		dlReq = dlReq.WithContext(auth.WithUserID(dlReq.Context(), userB))
		dlRR := httptest.NewRecorder()
		fileHandler.DownloadHandler(dlRR, dlReq)

		if dlRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 for downloading another user file, got %d", dlRR.Code)
		}
	})
}

func TestFileDownloadHandler_WithMetadata(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file_handler_meta_dl_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileHandler := handler.NewFileHandler(engine, 1024*1024)
	testUserID := uuid.New()

	// First upload a file to populate metadata in fileHandler
	content := []byte("Metadata attachment download test")
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "document.pdf")
	_, _ = part.Write(content)
	_ = writer.Close()

	uploadReq, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq = uploadReq.WithContext(auth.WithUserID(uploadReq.Context(), testUserID))
	uploadRR := httptest.NewRecorder()
	fileHandler.UploadHandler(uploadRR, uploadReq)

	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("upload failed: %d", uploadRR.Code)
	}

	var meta handler.FileMetadata
	if err := json.NewDecoder(uploadRR.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode upload response: %v", err)
	}

	// Now download the uploaded file
	dlReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/files/download/%s", meta.ID), nil)
	dlReq = dlReq.WithContext(auth.WithUserID(dlReq.Context(), testUserID))
	dlRR := httptest.NewRecorder()
	fileHandler.DownloadHandler(dlRR, dlReq)

	if dlRR.Code != http.StatusOK {
		t.Fatalf("download failed: %d", dlRR.Code)
	}

	disposition := dlRR.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "document.pdf") {
		t.Errorf("expected Content-Disposition to contain 'document.pdf', got %q", disposition)
	}

	// Also verify ListHandler returns this item
	listReq, _ := http.NewRequest(http.MethodGet, "/api/v1/files", nil)
	listReq = listReq.WithContext(auth.WithUserID(listReq.Context(), testUserID))
	listRR := httptest.NewRecorder()
	fileHandler.ListHandler(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("list failed: %d", listRR.Code)
	}

	var list []handler.FileMetadata
	if err := json.NewDecoder(listRR.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 file in list, got %d", len(list))
	}
}

func TestFileUploadHandler_SaveError500(t *testing.T) {
	unwritableEngine := storage.NewDiskEngine("/dev/null/invalid_path")
	unwritableHandler := handler.NewFileHandler(unwritableEngine, 1024*1024)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	_, _ = part.Write([]byte("data"))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(auth.WithUserID(req.Context(), uuid.New()))
	rr := httptest.NewRecorder()
	unwritableHandler.UploadHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 Internal Server Error, got %d", rr.Code)
	}
}
