package storage_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		{
			name:     "Path traversal dotdot in file ID returns error",
			baseDir:  "/storage",
			fileID:   "../etc/passwd",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Slash in file ID returns error",
			baseDir:  "/storage",
			fileID:   "sub/dir",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Backslash in file ID returns error",
			baseDir:  "/storage",
			fileID:   "foo\\bar",
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

type errReader struct{}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read failure")
}

func TestDiskStorageEngine_Save_ErrorPaths(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_err_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)

	t.Run("Invalid file ID returns error", func(t *testing.T) {
		_, _, err := engine.Save("f4", bytes.NewReader([]byte("data")), 1000)
		if err == nil {
			t.Error("expected error for short file ID, got nil")
		}
	})

	t.Run("Read error during streaming", func(t *testing.T) {
		fileID := "a1b2c3d4-1234-5678-9abc-def012345678"
		_, _, err := engine.Save(fileID, &errReader{}, 1000)
		if err == nil {
			t.Error("expected read error, got nil")
		}
	})
}

func TestDiskStorageEngine_GetFilePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage_get_path_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := storage.NewDiskEngine(tempDir)

	t.Run("Invalid file ID", func(t *testing.T) {
		_, err := engine.GetFilePath("ab")
		if !errors.Is(err, storage.ErrInvalidFileID) {
			t.Errorf("expected ErrInvalidFileID, got %v", err)
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := engine.GetFilePath("12345678-1234-5678-9abc-def012345678")
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})

	t.Run("Existing file", func(t *testing.T) {
		fileID := "e1e2e3e4-1234-5678-9abc-def012345678"
		content := []byte("existing file data")
		_, _, err := engine.Save(fileID, bytes.NewReader(content), 1000)
		if err != nil {
			t.Fatalf("failed to save file: %v", err)
		}

		path, err := engine.GetFilePath(fileID)
		if err != nil {
			t.Fatalf("expected path, got error: %v", err)
		}

		if path == "" {
			t.Error("expected non-empty path")
		}
	})
}

// TestDiskStorageEngine_EnsureStorageDir verifies startup storage directory initialization,
// probe write-and-remove verification, and error handling on unwritable paths (Task 1.1).
func TestDiskStorageEngine_EnsureStorageDir(t *testing.T) {
	t.Run("Successfully creates missing directory with 0775 and cleans up probe file", func(t *testing.T) {
		tempDir := t.TempDir()
		targetDir := filepath.Join(tempDir, "nested", "storage")

		engine := storage.NewDiskEngine(targetDir)
		err := engine.EnsureStorageDir()
		if err != nil {
			t.Fatalf("expected successful EnsureStorageDir, got: %v", err)
		}

		info, err := os.Stat(targetDir)
		if err != nil {
			t.Fatalf("failed to stat target directory: %v", err)
		}
		if !info.IsDir() {
			t.Fatalf("expected target path to be a directory")
		}
		if info.Mode().Perm() != 0775 {
			t.Errorf("expected directory permissions 0775, got %#o", info.Mode().Perm())
		}

		// Verify probe file was removed and cleaned up
		entries, err := os.ReadDir(targetDir)
		if err != nil {
			t.Fatalf("failed to read target directory: %v", err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".write-probe") {
				t.Errorf("residual probe file found: %s", entry.Name())
			}
		}
	})

	t.Run("Existing writable directory succeeds idempotently", func(t *testing.T) {
		tempDir := t.TempDir()
		engine := storage.NewDiskEngine(tempDir)

		if err := engine.EnsureStorageDir(); err != nil {
			t.Fatalf("first EnsureStorageDir failed: %v", err)
		}
		if err := engine.EnsureStorageDir(); err != nil {
			t.Fatalf("second EnsureStorageDir failed: %v", err)
		}

		entries, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".write-probe") {
				t.Errorf("residual probe file found: %s", entry.Name())
			}
		}
	})

	t.Run("Fails when directory is not writable", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping read-only directory test on windows")
		}
		tempDir := t.TempDir()
		roDir := filepath.Join(tempDir, "readonly")
		if err := os.Mkdir(roDir, 0555); err != nil {
			t.Fatalf("failed to create readonly dir: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(roDir, 0755)
		})

		engine := storage.NewDiskEngine(roDir)
		err := engine.EnsureStorageDir()
		if err == nil {
			t.Fatal("expected error on read-only storage directory, got nil")
		}
	})

	t.Run("Fails when target path is blocked by a file", func(t *testing.T) {
		tempDir := t.TempDir()
		regularFile := filepath.Join(tempDir, "regular-file")
		if err := os.WriteFile(regularFile, []byte("data"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		blockedDir := filepath.Join(regularFile, "subfolder")
		engine := storage.NewDiskEngine(blockedDir)
		err := engine.EnsureStorageDir()
		if err == nil {
			t.Fatal("expected error when directory creation is blocked by a file, got nil")
		}
	})
}

// TestDiskStorageEngine_Permissions verifies that Save creates shard parent directories
// with mode 0775 and saves binary files with mode 0664 for host user accessibility (Task 1.2).
func TestDiskStorageEngine_Permissions(t *testing.T) {
	tempDir := t.TempDir()
	engine := storage.NewDiskEngine(tempDir)
	fileID := "f47a8b90-1234-5678-9abc-def012345678"
	content := []byte("content for permissions verification")

	_, _, err := engine.Save(fileID, bytes.NewReader(content), 10000)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	savedPath := filepath.Join(tempDir, "f4", "7a", fileID)
	fileInfo, err := os.Stat(savedPath)
	if err != nil {
		t.Fatalf("failed to stat saved file: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0664 {
		t.Errorf("expected saved file permissions 0664, got %#o", perm)
	}

	shardL1 := filepath.Join(tempDir, "f4")
	l1Info, err := os.Stat(shardL1)
	if err != nil {
		t.Fatalf("failed to stat shard level 1 directory: %v", err)
	}
	if perm := l1Info.Mode().Perm(); perm != 0775 {
		t.Errorf("expected shard level 1 directory permissions 0775, got %#o", perm)
	}

	shardL2 := filepath.Join(tempDir, "f4", "7a")
	l2Info, err := os.Stat(shardL2)
	if err != nil {
		t.Fatalf("failed to stat shard level 2 directory: %v", err)
	}
	if perm := l2Info.Mode().Perm(); perm != 0775 {
		t.Errorf("expected shard level 2 directory permissions 0775, got %#o", perm)
	}
}
