## Why

In `file_upload.go`, the backend currently initiates a PostgreSQL database transaction (`tx.Begin()`) and locks the user record (`SELECT ... FOR UPDATE`) before reading multipart form data and streaming file contents to disk. Under slow client connections or large uploads, the database connection and row lock are held open for tens of seconds, leading to connection pool starvation and high latency. In addition, filenames longer than 255 characters cause unhandled PostgreSQL 500 errors instead of client-friendly 400 errors, folder names with null bytes are not rejected at input validation, orphan file removal on database failure is not logged, and test utilities in `file_upload_test.go` risk leaking temporary files on test failure.

## What Changes

- **Two-Phase Quota Pattern in File Upload:**
  - **Phase 1 (Optimistic Pre-Flight & Disk Stream):** Check quota optimistically without holding an open database transaction. Stream and write the incoming file to the sharded storage engine.
  - **Phase 2 (Short Atomic DB Transaction, ~1-3 ms):** Open a short transaction with `SELECT ... FOR UPDATE`, re-verify the remaining quota against the actual written size, insert file metadata, update user `used_bytes`, and commit. If the remaining quota was exceeded concurrently, rollback, remove the written file from disk with logging, and return HTTP 413 Payload Too Large.
- **Filename Length Validation:** Validate uploaded `header.Filename` length against a 255-character ceiling. Return HTTP 400 Bad Request if `len(header.Filename) > 255` or if the filename is empty.
- **Folder Name Null Byte Validation:** Reject folder creation requests where the folder name contains null bytes (`\x00`) with HTTP 400 Bad Request.
- **Logged Orphan File Cleanup:** Replace silent `_ = os.Remove(storagePath)` on SQL insert/commit failures with error logging (`log.Printf(...)`) to avoid masked cleanup errors.
- **Test File Cleanup Hardening & Test Suite Splitting:**
  - Add `t.Cleanup(func() { _ = os.Remove(tempFilePath) })` in `createMultipartRequestWithDiskTempFile` in `file_upload_test.go`.
  - Create focused test suite `file_upload_validation_test.go` to test filename length boundaries, folder null bytes, and two-phase quota concurrency without exceeding the repository's universal 450-line limit per file (`file_upload_test.go` is already 415 lines).

## Capabilities

### New Capabilities
*(None)*

### Modified Capabilities
- `file-storage`: Add Two-Phase Quota streaming pattern requirement, filename length validation limit (255 chars), and explicit logged compensation cleanup on persistence failure.
- `folder-management`: Add folder name input validation rule rejecting null bytes (`\x00`) with HTTP 400 Bad Request.

## Impact

- **Affected Backend Handlers:** `services/storage-service/internal/handler/file_upload.go`, `services/storage-service/internal/handler/folder.go`.
- **Affected Tests:** `services/storage-service/internal/handler/file_upload_test.go` (cleanup fix), new `file_upload_validation_test.go` (boundary & concurrency tests).
- **Database / API:** Database connection pool contention is drastically reduced during file uploads; HTTP 400 Bad Request is returned cleanly for oversized filenames (>255 chars) and null-byte folder names.
