package handler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

// FileMetadata represents a stored binary file's metadata record.
type FileMetadata struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}

// FileHandler manages HTTP file upload, download, and listing operations.
type FileHandler struct {
	engine       *storage.DiskEngine
	defaultQuota int64
	mu           sync.RWMutex
	files        map[string]FileMetadata
}

// NewFileHandler initializes a new FileHandler instance.
func NewFileHandler(engine *storage.DiskEngine, defaultQuota int64) *FileHandler {
	return &FileHandler{
		engine:       engine,
		defaultQuota: defaultQuota,
		files:        make(map[string]FileMetadata),
	}
}

// UploadHandler handles POST /api/v1/files/upload
func (fh *FileHandler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// Limit multipart header reading to 32MB
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing file field in form"})
		return
	}
	defer file.Close()

	fileID, err := generateUUID()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to generate file ID"})
		return
	}

	size, sha256Hex, err := fh.engine.Save(fileID, file, fh.defaultQuota)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if errors.Is(err, storage.ErrQuotaExceeded) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Storage quota exceeded"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to save file: %v", err)})
		return
	}

	meta := FileMetadata{
		ID:        fileID,
		UserID:    userID.String(),
		Filename:  header.Filename,
		Size:      size,
		SHA256:    sha256Hex,
		CreatedAt: time.Now(),
	}

	fh.mu.Lock()
	fh.files[fileID] = meta
	fh.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(meta)
}

// DownloadHandler handles GET /api/v1/files/download/:id
func (fh *FileHandler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	fileID := strings.TrimPrefix(r.URL.Path, "/api/v1/files/download/")
	fileID = strings.Trim(fileID, "/")
	if fileID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing file ID"})
		return
	}

	fh.mu.RLock()
	meta, exists := fh.files[fileID]
	fh.mu.RUnlock()

	if exists && meta.UserID != "" && meta.UserID != userID.String() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	filePath, err := fh.engine.GetFilePath(fileID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	f, err := os.Open(filePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to stat file"})
		return
	}

	filename := fileID
	if exists && meta.Filename != "" {
		filename = meta.Filename
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(filename)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.WriteHeader(http.StatusOK)

	_, _ = io.Copy(w, f)
}

// ListHandler handles GET /api/v1/files
func (fh *FileHandler) ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	fh.mu.RLock()
	list := make([]FileMetadata, 0)
	for _, meta := range fh.files {
		if meta.UserID == "" || meta.UserID == userID.String() {
			list = append(list, meta)
		}
	}
	fh.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(list)
}

func generateUUID() (string, error) {
	var buf [16]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}
