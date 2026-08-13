package storage_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RomanMischenko/SimpleCloud/services/storage-service/internal/storage"
)

// TestGetShardedPath tests the 2-level subfolder path generator (/storage/<uuid[0..1]>/<uuid[2..3]>/<uuid>)
func TestGetShardedPath(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		fileID   string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid UUID path generation",
			baseDir:  "/storage",
			fileID:   "f47a8b90-1234-5678-9abc-def012345678",
			expected: filepath.Join("/storage", "f4", "7a", "f47a8b90-1234-5678-9abc-def012345678"),
			wantErr:  false,
		},
		{
			name:     "Another valid UUID",
			baseDir:  "/var/data",
			fileID:   "01234567-89ab-cdef-0123-456789abcdef",
			expected: filepath.Join("/var/data", "01", "23", "01234567-89ab-cdef-0123-456789abcdef"),
			wantErr:  false,
		},
		{
			name:     "Short or invalid file ID",
			baseDir:  "/storage",
			fileID:   "f4",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := storage.GetShardedPath(tt.baseDir, tt.fileID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetShardedPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("GetShardedPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDiskStorageEngine_SaveAndQuota verifies writing files to sharded storage,
// calculating SHA256 on-the-fly, and aborting with cleanup when quota is exceeded.
func TestDiskStorageEngine_SaveAndQuota(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)
	fileID := "f47a8b90-1234-5678-9abc-def012345678"

	t.Run("Successful stream write with SHA256 calculation", func(t *testing.T) {
		content := []byte("Hello, SimpleCloud sharded storage engine!")
		expectedHashBytes := sha256.Sum256(content)
		expectedHash := hex.EncodeToString(expectedHashBytes[:])

		quotaLimit := int64(len(content) + 100)
		writtenBytes, sha256Hex, err := engine.Save(fileID, bytes.NewReader(content), quotaLimit)
		if err != nil {
			t.Fatalf("expected successful save, got err: %v", err)
		}

		if writtenBytes != int64(len(content)) {
			t.Errorf("expected written bytes %d, got %d", len(content), writtenBytes)
		}

		if sha256Hex != expectedHash {
			t.Errorf("expected SHA256 %s, got %s", expectedHash, sha256Hex)
		}

		// Verify file exists on disk at sharded path
		expectedPath := filepath.Join(tempDir, "f4", "7a", fileID)
		fileData, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("failed to read file from sharded path %s: %v", expectedPath, err)
		}

		if !bytes.Equal(fileData, content) {
			t.Errorf("file content mismatch. expected %q, got %q", content, fileData)
		}
	})

	t.Run("Quota breach aborts upload and cleans up partial temp file", func(t *testing.T) {
		overflowID := "01234567-89ab-cdef-0123-456789abcdef"
		content := bytes.Repeat([]byte("A"), 1000)
		strictQuota := int64(100) // Less than content size

		_, _, err := engine.Save(overflowID, bytes.NewReader(content), strictQuota)
		if err == nil {
			t.Fatal("expected quota exceeded error, got nil")
		}

		if !errors.Is(err, storage.ErrQuotaExceeded) {
			t.Errorf("expected ErrQuotaExceeded, got %v", err)
		}

		// Verify no partial file remains on disk
		expectedPath := filepath.Join(tempDir, "01", "23", overflowID)
		if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
			t.Errorf("expected file at %s to be deleted after quota breach, but it exists", expectedPath)
		}
	})
}
