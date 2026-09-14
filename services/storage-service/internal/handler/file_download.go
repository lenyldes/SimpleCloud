package handler

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/auth"
)

func init() {
	_ = mime.AddExtensionType(".mp4", "video/mp4")
	_ = mime.AddExtensionType(".m4v", "video/x-m4v")
	_ = mime.AddExtensionType(".webm", "video/webm")
	_ = mime.AddExtensionType(".mkv", "video/x-matroska")
	_ = mime.AddExtensionType(".avi", "video/x-msvideo")
	_ = mime.AddExtensionType(".mp3", "audio/mpeg")
	_ = mime.AddExtensionType(".ogg", "audio/ogg")
	_ = mime.AddExtensionType(".wav", "audio/wav")
	_ = mime.AddExtensionType(".flac", "audio/flac")
	_ = mime.AddExtensionType(".aac", "audio/aac")
	_ = mime.AddExtensionType(".txt", "text/plain; charset=utf-8")
	_ = mime.AddExtensionType(".pdf", "application/pdf")
	_ = mime.AddExtensionType(".json", "application/json")
	_ = mime.AddExtensionType(".csv", "text/csv; charset=utf-8")
	_ = mime.AddExtensionType(".png", "image/png")
	_ = mime.AddExtensionType(".jpg", "image/jpeg")
	_ = mime.AddExtensionType(".jpeg", "image/jpeg")
	_ = mime.AddExtensionType(".gif", "image/gif")
	_ = mime.AddExtensionType(".webp", "image/webp")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
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

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", formatContentDisposition(filename))

	http.ServeContent(w, r, filename, stat.ModTime(), f)
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
