package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

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
