## Context

Currently, `services/storage-service/internal/handler/file_upload.go` acquires a database connection, begins a transaction (`tx.Begin()`), and performs a blocking `SELECT ... FOR UPDATE` before processing the multipart upload stream and disk I/O (`fh.engine.Save()`). This creates long-lived database transaction locks during network transfers. Additionally, input sanitization lacks bounds for filename length (>255 characters) and folder name null bytes (`\x00`), and orphan file removal on persistence failure suppresses errors without diagnostic logging.

## Goals / Non-Goals

**Goals:**
- Implement the Two-Phase Quota Pattern for file uploads:
  - Phase 1 (Optimistic I/O): Non-transactional read of remaining quota, client stream processing, and disk file writing.
  - Phase 2 (Atomic Commit, 1-3 ms): Short database transaction acquiring `SELECT ... FOR UPDATE`, verifying remaining quota against final disk size, persisting file metadata, and incrementing `used_bytes`.
- Roll back the transaction and remove the saved disk file with error logging if the second phase detects a quota violation due to concurrent uploads.
- Add input validation rejecting filenames with length > 255 characters or empty names with HTTP 400 Bad Request.
- Add input validation in `folder.go` rejecting folder names containing null bytes (`\x00`) with HTTP 400 Bad Request.
- Log failures during orphan file removal (`os.Remove`) when database operations fail.
- Prevent temporary file leakage in `file_upload_test.go` by adding `t.Cleanup`.
- Proactively isolate new tests into `file_upload_validation_test.go` to strictly satisfy the universal <= 450 lines file limit.

**Non-Goals:**
- Changing the disk storage sharding scheme or path hierarchy.
- Modifying frontend upload progress UI or chunked multipart client protocols.
- Refactoring `FolderHandler` beyond input validation.

## Decisions

### Decision 1: Two-Phase Quota Pattern
- **Approach:**
  - *Phase 1:* Read `used_bytes` and `quota_bytes` from PostgreSQL without a transaction (`fh.pool.QueryRow`). Compute remaining quota. If non-positive or less than `Content-Length`, reject immediately with HTTP 413. Stream the file to disk using `fh.engine.Save(fileID, file, remainingQuota)`.
  - *Phase 2:* Open a database transaction (`tx.Begin`), lock the user row (`SELECT ... FOR UPDATE`), check if `usedBytes + size <= quotaBytes`. If exceeded, rollback `tx`, delete the disk file via `os.Remove`, log if removal fails, and return HTTP 413. Otherwise insert file metadata, update user `used_bytes`, and commit `tx`.
- **Alternatives Considered:**
  - *Reserve-and-Commit:* Temporarily incrementing `used_bytes` before upload and rolling back on disconnect. Rejected due to complexity in handling crashed servers, orphan reservations, and lease timeouts.
  - *Single Long Transaction (status quo):* Rejected because holding row locks across network I/O exhausts PostgreSQL connection pools under slow uploads.

### Decision 2: Input Validation at Handler Boundary
- Filename length: Validate `len(header.Filename) > 255 || header.Filename == ""` immediately after parsing the form file header, returning HTTP 400 Bad Request with an explicit error JSON before any disk writing.
- Folder name null bytes: In `folder.go:CreateHandler`, extend the forbidden character check with `strings.Contains(trimmedName, "\x00")` returning HTTP 400 Bad Request.

### Decision 3: Logged Orphan File Cleanup
- Replace silent suppression `_ = os.Remove(storagePath)` with:
  ```go
  if rmErr := os.Remove(storagePath); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
      log.Printf("file_upload: failed to clean up orphan storage file %s: %v", storagePath, rmErr)
  }
  ```
  ensuring administrators and audit logs have visibility into filesystem inconsistencies.

### Decision 4: Modular Test File Separation (<= 450 lines limit)
- `file_upload_test.go` is currently at 415 lines. Adding new tests directly would breach the repository's universal 450-line ceiling.
- New tests for filename length limits, folder null byte validation, orphan cleanup logging, and concurrent two-phase quota races will reside in a new test file: `services/storage-service/internal/handler/file_upload_validation_test.go`.

## Risks / Trade-offs

- **[Risk] Optimistic upload writes file to disk even if concurrent upload exhausts quota before commit.**  
  → *Mitigation:* The atomic Phase 2 strictly verifies quota under `SELECT ... FOR UPDATE`. If quota is exhausted, the written file is immediately removed from disk, ensuring no quota overrun or disk leaks.
- **[Risk] Client disconnects during Phase 1 disk write.**  
  → *Mitigation:* `fh.engine.Save` already handles write errors and cleans up partial files; no database lock was held, so database health is unaffected.
