package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

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

	r.Body = http.MaxBytesReader(w, r.Body, remainingQuota+1024*1024)

	// Limit multipart header reading to 32MB
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "payload too large"})
			return
		}
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
