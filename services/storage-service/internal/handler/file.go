package handler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

// FileMetadata represents a stored binary file's metadata record.
type FileMetadata struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	FolderID  *string   `json:"folder_id,omitempty"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}

// FileHandler manages HTTP file upload, download, listing, and deletion.
type FileHandler struct {
	engine       *storage.DiskEngine
	pool         *pgxpool.Pool
	defaultQuota int64
}

// NewFileHandler initializes a new FileHandler instance with PostgreSQL pool.
func NewFileHandler(engine *storage.DiskEngine, pool *pgxpool.Pool, defaultQuota int64) *FileHandler {
	return &FileHandler{
		engine:       engine,
		pool:         pool,
		defaultQuota: defaultQuota,
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

	if fh.pool == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "database pool unavailable"})
		return
	}

	tx, err := fh.pool.Begin(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var usedBytes, quotaBytes int64
	err = tx.QueryRow(r.Context(),
		`SELECT used_bytes, quota_bytes FROM users WHERE id = $1 FOR UPDATE`,
		userID,
	).Scan(&usedBytes, &quotaBytes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user record not found"})
		return
	}

	remainingQuota := quotaBytes - usedBytes
	if remainingQuota < 0 {
		remainingQuota = 0
	}

	if r.Header.Get("Content-Length") != "" {
		if contentLength, err := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64); err == nil {
			if contentLength > remainingQuota {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Storage quota exceeded"})
				return
			}
		}
	}

	// Limit multipart header reading to 32MB
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid multipart form"})
		return
	}

	folderIDVal := r.FormValue("folder_id")
	var folderIDPtr *string
	var folderUUID *uuid.UUID
	if folderIDVal != "" {
		folderIDPtr = &folderIDVal
		if parsed, parseErr := uuid.Parse(folderIDVal); parseErr == nil {
			folderUUID = &parsed
		}
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

	size, sha256Hex, err := fh.engine.Save(fileID, file, remainingQuota)
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

	storagePath, _ := fh.engine.GetFilePath(fileID)

	var dbFolderID interface{} = nil
	if folderUUID != nil {
		dbFolderID = *folderUUID
	} else if folderIDVal != "" {
		dbFolderID = folderIDVal
	}

	createdAt := time.Now()
	_, err = tx.Exec(r.Context(),
		`INSERT INTO files (id, user_id, folder_id, filename, size_bytes, sha256_hash, storage_path, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fileID, userID, dbFolderID, header.Filename, size, sha256Hex, storagePath, createdAt,
	)
	if err != nil {
		_ = os.Remove(storagePath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to save metadata: %v", err)})
		return
	}

	_, err = tx.Exec(r.Context(),
		`UPDATE users SET used_bytes = used_bytes + $1 WHERE id = $2`,
		size, userID,
	)
	if err != nil {
		_ = os.Remove(storagePath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to update user quota: %v", err)})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		_ = os.Remove(storagePath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to commit metadata: %v", err)})
		return
	}

	meta := FileMetadata{
		ID:        fileID,
		UserID:    userID.String(),
		FolderID:  folderIDPtr,
		Filename:  header.Filename,
		Size:      size,
		SHA256:    sha256Hex,
		CreatedAt: createdAt,
	}

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

	if fh.pool == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	var ownerID uuid.UUID
	var filename, storagePath string
	err := fh.pool.QueryRow(r.Context(),
		`SELECT user_id, filename, storage_path FROM files WHERE id = $1`,
		fileID,
	).Scan(&ownerID, &filename, &storagePath)
	if err != nil || ownerID != userID {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	f, err := os.Open(storagePath)
	if err != nil {
		if diskPath, err2 := fh.engine.GetFilePath(fileID); err2 == nil {
			f, err = os.Open(diskPath)
		}
	}
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

	list := make([]FileMetadata, 0)
	if fh.pool == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(list)
		return
	}

	hasFolderID := r.URL.Query().Has("folder_id")
	targetFolderID := r.URL.Query().Get("folder_id")

	var rows pgx.Rows
	var err error

	if !hasFolderID || targetFolderID == "" {
		rows, err = fh.pool.Query(r.Context(),
			`SELECT id, user_id, folder_id, filename, size_bytes, sha256_hash, created_at
			 FROM files WHERE user_id = $1 AND folder_id IS NULL ORDER BY created_at DESC`,
			userID,
		)
	} else {
		parsedFolderID, parseErr := uuid.Parse(targetFolderID)
		if parseErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(list)
			return
		}
		rows, err = fh.pool.Query(r.Context(),
			`SELECT id, user_id, folder_id, filename, size_bytes, sha256_hash, created_at
			 FROM files WHERE user_id = $1 AND folder_id = $2 ORDER BY created_at DESC`,
			userID, parsedFolderID,
		)
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to query files"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, dbUserID string
		var folderID *uuid.UUID
		var filename, sha256Hash string
		var size int64
		var createdAt time.Time

		if err := rows.Scan(&id, &dbUserID, &folderID, &filename, &size, &sha256Hash, &createdAt); err != nil {
			continue
		}
		var folderIDStr *string
		if folderID != nil {
			s := folderID.String()
			folderIDStr = &s
		}
		list = append(list, FileMetadata{
			ID:        id,
			UserID:    dbUserID,
			FolderID:  folderIDStr,
			Filename:  filename,
			Size:      size,
			SHA256:    sha256Hash,
			CreatedAt: createdAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(list)
}

// DeleteHandler handles DELETE /api/v1/files/:id
func (fh *FileHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	fileID := strings.TrimPrefix(r.URL.Path, "/api/v1/files/")
	fileID = strings.Trim(fileID, "/")
	if fileID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing file ID"})
		return
	}

	parsedFileID, err := uuid.Parse(fileID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	if fh.pool == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	var size int64
	var storagePath string
	err = fh.pool.QueryRow(r.Context(),
		`SELECT size_bytes, storage_path FROM files WHERE id = $1 AND user_id = $2`,
		parsedFileID, userID,
	).Scan(&size, &storagePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file not found"})
		return
	}

	tx, err := fh.pool.Begin(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	_, err = tx.Exec(r.Context(), `DELETE FROM files WHERE id = $1 AND user_id = $2`, parsedFileID, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete file metadata"})
		return
	}

	_, err = tx.Exec(r.Context(), `UPDATE users SET used_bytes = GREATEST(used_bytes - $1, 0) WHERE id = $2`, size, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to update user quota"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to commit file deletion"})
		return
	}

	if err := os.Remove(storagePath); err != nil {
		log.Printf("Warning: failed to remove deleted file binary from disk: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
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
