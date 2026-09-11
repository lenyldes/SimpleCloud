package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

// FolderMetadata represents folder details in JSON responses.
type FolderMetadata struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	ParentID  *string `json:"parent_id,omitempty"`
	Name      string  `json:"name"`
	CreatedAt string  `json:"created_at,omitempty"`
}

// FolderHandler handles HTTP endpoints for directory management.
type FolderHandler struct {
	pool   *pgxpool.Pool
	engine *storage.DiskEngine
}

// NewFolderHandler initializes a typed FolderHandler instance.
func NewFolderHandler(pool *pgxpool.Pool, engine *storage.DiskEngine) *FolderHandler {
	return &FolderHandler{
		pool:   pool,
		engine: engine,
	}
}

func writeFolderJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// CreateHandler handles POST /api/v1/folders
func (fh *FolderHandler) CreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		writeFolderJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeFolderJSONError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	trimmedName := strings.TrimSpace(req.Name)
	if trimmedName == "" {
		writeFolderJSONError(w, http.StatusBadRequest, "folder name cannot be empty")
		return
	}

	var parentUUID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		parsed, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeFolderJSONError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		parentUUID = &parsed

		if fh.pool != nil {
			var dummy int
			err := fh.pool.QueryRow(r.Context(),
				`SELECT 1 FROM folders WHERE id = $1 AND user_id = $2`,
				parsed, userID,
			).Scan(&dummy)
			if err != nil {
				writeFolderJSONError(w, http.StatusNotFound, "parent folder not found")
				return
			}
		}
	}

	folderID := uuid.New().String()
	createdAtStr := time.Now().Format(time.RFC3339)

	if fh.pool != nil {
		var dbParentID interface{} = nil
		if parentUUID != nil {
			dbParentID = *parentUUID
		}
		_, err := fh.pool.Exec(r.Context(),
			`INSERT INTO folders (id, user_id, parent_id, name, created_at) VALUES ($1, $2, $3, $4, NOW())`,
			folderID, userID, dbParentID, trimmedName,
		)
		if err != nil {
			writeFolderJSONError(w, http.StatusInternalServerError, "failed to create folder record")
			return
		}
	}

	meta := FolderMetadata{
		ID:        folderID,
		UserID:    userID.String(),
		ParentID:  req.ParentID,
		Name:      trimmedName,
		CreatedAt: createdAtStr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(meta)
}

// ListHandler handles GET /api/v1/folders
func (fh *FolderHandler) ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		writeFolderJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	result := make([]FolderMetadata, 0)
	if fh.pool == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
		return
	}

	allParam := r.URL.Query().Get("all") == "true"
	hasParentID := r.URL.Query().Has("parent_id")
	targetParentID := r.URL.Query().Get("parent_id")

	var rows pgx.Rows
	var err error

	if allParam {
		rows, err = fh.pool.Query(r.Context(),
			`SELECT id, user_id, parent_id, name, created_at FROM folders WHERE user_id = $1 ORDER BY created_at ASC`,
			userID,
		)
	} else if !hasParentID || targetParentID == "" {
		rows, err = fh.pool.Query(r.Context(),
			`SELECT id, user_id, parent_id, name, created_at FROM folders WHERE user_id = $1 AND parent_id IS NULL ORDER BY created_at ASC`,
			userID,
		)
	} else {
		parsedParentID, parseErr := uuid.Parse(targetParentID)
		if parseErr != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(result)
			return
		}
		rows, err = fh.pool.Query(r.Context(),
			`SELECT id, user_id, parent_id, name, created_at FROM folders WHERE user_id = $1 AND parent_id = $2 ORDER BY created_at ASC`,
			userID, parsedParentID,
		)
	}

	if err != nil {
		writeFolderJSONError(w, http.StatusInternalServerError, "failed to query folders")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, dbUserID, name string
		var parentID *uuid.UUID
		var createdAt time.Time

		if err := rows.Scan(&id, &dbUserID, &parentID, &name, &createdAt); err != nil {
			log.Printf("ListHandler: failed to scan folder: %v", err)
			writeFolderJSONError(w, http.StatusInternalServerError, "failed to scan folder")
			return
		}
		var parentIDStr *string
		if parentID != nil {
			s := parentID.String()
			parentIDStr = &s
		}
		result = append(result, FolderMetadata{
			ID:        id,
			UserID:    dbUserID,
			ParentID:  parentIDStr,
			Name:      name,
			CreatedAt: createdAt.Format(time.RFC3339),
		})
	}

	if err := rows.Err(); err != nil {
		writeFolderJSONError(w, http.StatusInternalServerError, "failed reading folders")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// DeleteHandler handles DELETE /api/v1/folders/:id
func (fh *FolderHandler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		writeFolderJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	folderID := strings.TrimPrefix(r.URL.Path, "/api/v1/folders/")
	folderID = strings.Trim(folderID, "/")
	if folderID == "" {
		writeFolderJSONError(w, http.StatusBadRequest, "missing folder ID")
		return
	}

	parsedFolderID, err := uuid.Parse(folderID)
	if err != nil {
		writeFolderJSONError(w, http.StatusBadRequest, "invalid folder ID")
		return
	}

	if fh.pool == nil {
		writeFolderJSONError(w, http.StatusNotFound, "folder not found")
		return
	}

	var dummy int
	err = fh.pool.QueryRow(r.Context(),
		`SELECT 1 FROM folders WHERE id = $1 AND user_id = $2`,
		parsedFolderID, userID,
	).Scan(&dummy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeFolderJSONError(w, http.StatusNotFound, "folder not found")
			return
		}
		writeFolderJSONError(w, http.StatusInternalServerError, "database error")
		return
	}

	tx, err := fh.pool.Begin(r.Context())
	if err != nil {
		writeFolderJSONError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// Step 1: collect file paths, file IDs, and sizes in the subtree
	queryCollectFiles := `
	WITH RECURSIVE subfolders AS (
		SELECT id FROM folders WHERE id = $1 AND user_id = $2
		UNION ALL
		SELECT f.id FROM folders f
		JOIN subfolders sf ON f.parent_id = sf.id
	)
	SELECT fi.id, fi.storage_path, fi.size_bytes
	FROM files fi
	WHERE fi.folder_id IN (SELECT id FROM subfolders);
	`
	fileRows, err := tx.Query(r.Context(), queryCollectFiles, parsedFolderID, userID)
	if err != nil {
		writeFolderJSONError(w, http.StatusInternalServerError, "failed to collect folder files")
		return
	}
	defer fileRows.Close()

	type fileToDelete struct {
		id          string
		storagePath string
		size        int64
	}
	var filesToDelete []fileToDelete
	var totalSize int64

	for fileRows.Next() {
		var f fileToDelete
		if err := fileRows.Scan(&f.id, &f.storagePath, &f.size); err != nil {
			_ = tx.Rollback(r.Context())
			writeFolderJSONError(w, http.StatusInternalServerError, "failed to scan folder files")
			return
		}
		filesToDelete = append(filesToDelete, f)
		totalSize += f.size
	}
	if err := fileRows.Err(); err != nil {
		_ = tx.Rollback(r.Context())
		writeFolderJSONError(w, http.StatusInternalServerError, "failed reading folder files")
		return
	}
	fileRows.Close()

	// Step 2: DELETE files in subfolders tree
	queryDeleteFiles := `
	WITH RECURSIVE subfolders AS (
		SELECT id FROM folders WHERE id = $1 AND user_id = $2
		UNION ALL
		SELECT f.id FROM folders f
		JOIN subfolders sf ON f.parent_id = sf.id
	)
	DELETE FROM files WHERE folder_id IN (SELECT id FROM subfolders);
	`
	_, err = tx.Exec(r.Context(), queryDeleteFiles, parsedFolderID, userID)
	if err != nil {
		writeFolderJSONError(w, http.StatusInternalServerError, "failed to delete subfolder files")
		return
	}

	// Step 3: DELETE folders in subfolders tree
	queryDeleteFolders := `
	WITH RECURSIVE subfolders AS (
		SELECT id FROM folders WHERE id = $1 AND user_id = $2
		UNION ALL
		SELECT f.id FROM folders f
		JOIN subfolders sf ON f.parent_id = sf.id
	)
	DELETE FROM folders WHERE id IN (SELECT id FROM subfolders);
	`
	_, err = tx.Exec(r.Context(), queryDeleteFolders, parsedFolderID, userID)
	if err != nil {
		writeFolderJSONError(w, http.StatusInternalServerError, "failed to delete subfolders")
		return
	}

	// Step 4: UPDATE user used_bytes with clamp
	if totalSize > 0 {
		_, err = tx.Exec(r.Context(),
			`UPDATE users SET used_bytes = GREATEST(used_bytes - $1, 0) WHERE id = $2`,
			totalSize, userID,
		)
		if err != nil {
			writeFolderJSONError(w, http.StatusInternalServerError, "failed to update user quota")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeFolderJSONError(w, http.StatusInternalServerError, "failed to commit folder deletion")
		return
	}

	// Physical disk cleanup best-effort
	for _, f := range filesToDelete {
		if err := os.Remove(f.storagePath); err != nil {
			log.Printf("Warning: failed to remove file shard from disk: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
