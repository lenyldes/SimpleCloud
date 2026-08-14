package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	pool    *pgxpool.Pool
	engine  *storage.DiskEngine
	mu      sync.RWMutex
	folders map[string]FolderMetadata
}

// NewFolderHandler initializes a FolderHandler instance.
func NewFolderHandler(args ...interface{}) *FolderHandler {
	fh := &FolderHandler{
		folders: make(map[string]FolderMetadata),
	}
	for _, arg := range args {
		switch v := arg.(type) {
		case *pgxpool.Pool:
			fh.pool = v
		case *storage.DiskEngine:
			fh.engine = v
		}
	}
	return fh
}

// CreateHandler handles POST /api/v1/folders
func (fh *FolderHandler) CreateHandler(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON payload"})
		return
	}

	trimmedName := strings.TrimSpace(req.Name)
	if trimmedName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "folder name cannot be empty"})
		return
	}

	fh.mu.Lock()
	defer fh.mu.Unlock()

	if req.ParentID != nil && *req.ParentID != "" {
		parent, exists := fh.folders[*req.ParentID]
		if !exists || parent.UserID != userID.String() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "parent folder not found"})
			return
		}
	}

	folderID := uuid.New().String()
	meta := FolderMetadata{
		ID:        folderID,
		UserID:    userID.String(),
		ParentID:  req.ParentID,
		Name:      trimmedName,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	if fh.pool != nil {
		var parentUUID *uuid.UUID
		if req.ParentID != nil && *req.ParentID != "" {
			parsed, err := uuid.Parse(*req.ParentID)
			if err == nil {
				parentUUID = &parsed
			}
		}
		_, err := fh.pool.Exec(r.Context(),
			`INSERT INTO folders (id, user_id, parent_id, name) VALUES ($1, $2, $3, $4)`,
			folderID, userID, parentUUID, trimmedName,
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create folder record"})
			return
		}
	}

	fh.folders[folderID] = meta

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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	hasParentID := r.URL.Query().Has("parent_id")
	targetParentID := r.URL.Query().Get("parent_id")

	fh.mu.RLock()
	defer fh.mu.RUnlock()

	result := make([]FolderMetadata, 0)

	for _, f := range fh.folders {
		if f.UserID != userID.String() {
			continue
		}
		if hasParentID {
			if targetParentID == "" {
				if f.ParentID == nil || *f.ParentID == "" {
					result = append(result, f)
				}
			} else {
				if f.ParentID != nil && *f.ParentID == targetParentID {
					result = append(result, f)
				}
			}
		} else {
			if f.ParentID == nil || *f.ParentID == "" {
				result = append(result, f)
			}
		}
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	folderID := strings.TrimPrefix(r.URL.Path, "/api/v1/folders/")
	folderID = strings.Trim(folderID, "/")
	if folderID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing folder ID"})
		return
	}

	fh.mu.Lock()
	defer fh.mu.Unlock()

	folder, exists := fh.folders[folderID]
	if !exists || folder.UserID != userID.String() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "folder not found"})
		return
	}

	subfolderIDs := fh.collectSubfoldersLocked(folderID)
	subfolderIDs = append(subfolderIDs, folderID)

	for _, id := range subfolderIDs {
		delete(fh.folders, id)
	}

	if fh.pool != nil {
		_, _ = fh.pool.Exec(r.Context(), `DELETE FROM folders WHERE id = $1 AND user_id = $2`, folderID, userID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (fh *FolderHandler) collectSubfoldersLocked(parentID string) []string {
	var ids []string
	for id, f := range fh.folders {
		if f.ParentID != nil && *f.ParentID == parentID {
			ids = append(ids, id)
			ids = append(ids, fh.collectSubfoldersLocked(id)...)
		}
	}
	return ids
}
