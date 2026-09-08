package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/handler"
)

func decodeFileMeta(t *testing.T, rr *httptest.ResponseRecorder) handler.FileMetadata {
	t.Helper()
	var meta handler.FileMetadata
	if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode file metadata JSON: %v", err)
	}
	return meta
}

func decodeFileList(t *testing.T, rr *httptest.ResponseRecorder) []handler.FileMetadata {
	t.Helper()
	var list []handler.FileMetadata
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode file list JSON: %v", err)
	}
	return list
}

func listFilesRequest(fh *handler.FileHandler, userID uuid.UUID, url string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.ListHandler(rr, req)
	return rr
}

func downloadFileRequest(fh *handler.FileHandler, userID uuid.UUID, fileID string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodGet, "/api/v1/files/download/"+fileID, nil)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.DownloadHandler(rr, req)
	return rr
}

func deleteFileRequest(fh *handler.FileHandler, userID uuid.UUID, fileID string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodDelete, "/api/v1/files/"+fileID, nil)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()
	fh.DeleteHandler(rr, req)
	return rr
}
