## 1. Test Preparation & RED State Tests

- [x] 1.1 In `file_upload_test.go`, add `t.Cleanup(func() { _ = os.Remove(tempFilePath) })` inside `createMultipartRequestWithDiskTempFile` and verify existing tests compile and pass.
- [x] 1.2 In `file_upload_validation_test.go`, write failing unit tests for filename validation (filenames > 255 characters or empty returning HTTP 400 Bad Request) and folder name null byte validation (`\x00` returning HTTP 400 Bad Request).
- [x] 1.3 In `file_upload_validation_test.go`, write failing tests for the Two-Phase Quota Pattern: testing concurrent upload race conditions, rollback and logged orphan file cleanup on SQL/quota failures, and verify tests fail in RED state while keeping file length <= 450 lines.

## 2. Production Implementation & GREEN State Verification

- [x] 2.1 In `folder.go:CreateHandler`, add null byte check `strings.Contains(trimmedName, "\x00")` returning HTTP 400 Bad Request and verify folder null byte test passes.
- [x] 2.2 In `file_upload.go:UploadHandler`, add filename validation ensuring `header.Filename != ""` and `len(header.Filename) <= 255` returning HTTP 400 Bad Request, verifying filename validation tests pass.
- [x] 2.3 In `file_upload.go:UploadHandler`, refactor upload flow to Two-Phase Quota:
  - Phase 1: Query remaining quota non-transactionally, perform pre-flight checks, read multipart form, and stream file to disk without holding a DB transaction.
  - Phase 2: Open a short atomic transaction, execute `SELECT ... FOR UPDATE`, verify quota against written size, insert metadata, increment `used_bytes`, and commit.
- [x] 2.4 In `file_upload.go:UploadHandler`, ensure all compensation `os.Remove(storagePath)` calls on SQL/quota errors log removal errors via `log.Printf`.
- [x] 2.5 Run `go test -v -race -cover ./...` in `services/storage-service/internal/handler`, verify all tests pass with >= 85% coverage and all files remain strictly <= 450 lines.
