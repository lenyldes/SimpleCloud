package handler

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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
