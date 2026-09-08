package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

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

	if _, err := uuid.Parse(fileID); err != nil {
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

	w.Header().Set("Content-Disposition", formatContentDisposition(filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.WriteHeader(http.StatusOK)

	_, _ = io.Copy(w, f)
}

// formatContentDisposition constructs Content-Disposition with RFC 5987 filename* UTF-8 encoding
// and sanitized ASCII fallback to prevent header injection vulnerabilities.
func formatContentDisposition(filename string) string {
	base := filepath.Base(filename)
	sanitized := strings.NewReplacer("\r", "", "\n", "", "\"", "").Replace(base)
	if sanitized == "" {
		sanitized = "file"
	}

	var asciiBuf strings.Builder
	hasNonASCII := false
	for i := 0; i < len(sanitized); i++ {
		b := sanitized[i]
		if b > 127 {
			hasNonASCII = true
		} else if b >= 32 {
			asciiBuf.WriteByte(b)
		}
	}

	asciiFallback := strings.TrimSpace(asciiBuf.String())
	if asciiFallback == "" {
		asciiFallback = "file"
	}

	if hasNonASCII {
		encoded := url.PathEscape(sanitized)
		return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", asciiFallback, encoded)
	}

	return fmt.Sprintf("attachment; filename=%q", asciiFallback)
}
