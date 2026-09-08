package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid file ID"})
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
