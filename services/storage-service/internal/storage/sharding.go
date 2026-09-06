package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrQuotaExceeded is returned when an upload exceeds available user quota.
	ErrQuotaExceeded = errors.New("storage quota exceeded")

	// ErrInvalidFileID is returned when file ID is invalid or too short for sharding.
	ErrInvalidFileID = errors.New("invalid file ID length for sharding")
)

// GetShardedPath calculates 2-level subfolder path: /baseDir/<uuid[0..1]>/<uuid[2..3]>/<uuid>
func GetShardedPath(baseDir string, fileID string) (string, error) {
	if len(fileID) < 4 || strings.ContainsAny(fileID, "/\\") || filepath.Base(fileID) != fileID {
		return "", ErrInvalidFileID
	}
	sub1 := fileID[0:2]
	sub2 := fileID[2:4]
	return filepath.Join(baseDir, sub1, sub2, fileID), nil
}

// DiskEngine handles binary file storage on disk with subfolder sharding and quota checks.
type DiskEngine struct {
	baseDir string
}

// NewDiskEngine creates a new disk storage engine targeting baseDir.
func NewDiskEngine(baseDir string) *DiskEngine {
	return &DiskEngine{
		baseDir: baseDir,
	}
}

// EnsureStorageDir verifies that the base storage directory exists and is writable.
// If missing, it creates the directory with mode 0775. Write access is verified
// via a probe file creation and cleanup check.
func (e *DiskEngine) EnsureStorageDir() error {
	info, err := os.Stat(e.baseDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(e.baseDir, 0775); err != nil {
			return fmt.Errorf("failed to create storage directory %s: %w", e.baseDir, err)
		}
		if err := os.Chmod(e.baseDir, 0775); err != nil {
			return fmt.Errorf("failed to set storage directory permissions %s: %w", e.baseDir, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat storage directory %s: %w", e.baseDir, err)
	} else if !info.IsDir() {
		return fmt.Errorf("storage path %s is not a directory", e.baseDir)
	}

	probeFile, err := os.CreateTemp(e.baseDir, ".write-probe-*")
	if err != nil {
		return fmt.Errorf("storage directory %s is not writable: %w", e.baseDir, err)
	}
	probePath := probeFile.Name()
	_ = probeFile.Close()
	if err := os.Remove(probePath); err != nil {
		return fmt.Errorf("failed to remove write probe file %s: %w", probePath, err)
	}

	return nil
}

// Save streams data from reader r into sharded storage path, computing SHA256 on-the-fly
// and enforcing quota limit. Partial files are deleted on error or quota breach.
func (e *DiskEngine) Save(fileID string, r io.Reader, quotaLimit int64) (int64, string, error) {
	targetPath, err := GetShardedPath(e.baseDir, fileID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get sharded path: %w", err)
	}

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0775); err != nil {
		return 0, "", fmt.Errorf("failed to create directory structure %s: %w", dir, err)
	}

	sub1 := filepath.Join(e.baseDir, fileID[0:2])
	_ = os.Chmod(sub1, 0775)
	_ = os.Chmod(dir, 0775)

	tempFile, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return 0, "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure temp file cleanup on error
	var success bool
	defer func() {
		_ = tempFile.Close()
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	hasher := sha256.New()
	buf := make([]byte, 32*1024)
	var totalWritten int64

	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			if quotaLimit > 0 && (totalWritten+int64(n)) > quotaLimit {
				return 0, "", ErrQuotaExceeded
			}

			wn, writeErr := tempFile.Write(buf[:n])
			if writeErr != nil {
				return 0, "", fmt.Errorf("failed to write to temp file: %w", writeErr)
			}
			hasher.Write(buf[:wn])
			totalWritten += int64(wn)
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return 0, "", fmt.Errorf("read error during streaming: %w", readErr)
		}
	}

	_ = tempFile.Close()

	if err := os.Rename(tempPath, targetPath); err != nil {
		return 0, "", fmt.Errorf("failed to move temp file to target path %s: %w", targetPath, err)
	}

	if err := os.Chmod(targetPath, 0664); err != nil {
		return 0, "", fmt.Errorf("failed to set permissions on saved file %s: %w", targetPath, err)
	}

	success = true
	sha256Hex := hex.EncodeToString(hasher.Sum(nil))
	return totalWritten, sha256Hex, nil
}

// GetFilePath returns full path to a file if it exists.
func (e *DiskEngine) GetFilePath(fileID string) (string, error) {
	path, err := GetShardedPath(e.baseDir, fileID)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}
