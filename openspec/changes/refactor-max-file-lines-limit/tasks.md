## 1. Test Suite & Architecture Guard (RED Phase - [TEST-AGENT])

- [x] 1.1 Implement `TestMaxFileLineCount` in `services/storage-service/cmd/architecture_test.go` scanning repository files via `git ls-files` with double-check binary exclusion (known binary extensions + 4KB null-byte `0x00` and invalid UTF-8 check) enforcing `<= 450` lines ceiling.
- [x] 1.2 Verify `TestMaxFileLineCount` fails as expected (RED state) listing the 5 known violating files (`file_test.go`, `app.js`, `folder_test.go`, `file.go`, `handler_test.go`).

## 2. Backend Handler & Test Decomposition (GREEN Phase - [CODE-AGENT])

- [x] 2.1 Decompose `services/storage-service/internal/handler/file.go` (534 lines) into `file.go` (structs, `NewFileHandler`, common response helpers), `file_upload.go` (`UploadFile`), `file_download.go` (`DownloadFile`), `file_list.go` (`ListFiles`), and `file_delete.go` (`DeleteFile`), ensuring each file is <= 450 lines and passes compilation.
- [x] 2.2 Decompose `services/storage-service/internal/handler/file_test.go` (1043 lines) into `file_upload_test.go`, `file_download_test.go`, `file_list_test.go`, and `file_delete_test.go`, ensuring each file is <= 450 lines.
- [x] 2.3 Decompose `services/storage-service/internal/handler/folder_test.go` (629 lines) into `folder_crud_test.go`, `folder_hierarchy_test.go`, and `folder_delete_test.go`, ensuring each file is <= 450 lines.
- [x] 2.4 Decompose `services/storage-service/internal/auth/handler_test.go` (479 lines) into `handler_auth_test.go` and `handler_session_test.go`, ensuring each file is <= 450 lines.
- [x] 2.5 Run `go test -v -cover ./...` and verify that all backend handler tests pass cleanly with >= 85% statement coverage in `internal/*`.

## 3. Frontend JavaScript Zero-Build Modularization (GREEN Phase - [CODE-AGENT])

- [x] 3.1 Extract API and network methods into `services/web-frontend/src/js/api.js` (including 401 interceptor `fetchWithAuth`) ensuring <= 450 lines.
- [x] 3.2 Extract authentication modal logic, session check, profile dropdown, and logout handler into `services/web-frontend/src/js/auth.js` ensuring <= 450 lines.
- [x] 3.3 Extract DOM rendering for file grid/list, breadcrumbs, quota progress bar, and toasts into `services/web-frontend/src/js/ui.js` ensuring <= 450 lines.
- [x] 3.4 Extract lightbox preview, code/text viewer, video player, and new folder modal logic into `services/web-frontend/src/js/modals.js` ensuring <= 450 lines.
- [x] 3.5 Refactor `services/web-frontend/src/js/app.js` into a lightweight orchestrator (`init`, event listeners, dropzone binding, state bootstrap) under 450 lines and remove legacy monolithic `services/web-frontend/src/app.js`.
- [x] 3.6 Update `services/web-frontend/src/index.html` script tags to load the modular scripts in dependency order (`api.js`, `auth.js`, `ui.js`, `modals.js`, `app.js`) with cache-busting version query strings.

## 4. Verification & Architecture Guard Validation (GREEN Phase - [CODE-AGENT])

- [x] 4.1 Run `go test -v -run TestMaxFileLineCount ./cmd` to verify `TestMaxFileLineCount` passes cleanly (GREEN state) with zero file violations.
- [x] 4.2 Run complete test suite `go test -v -cover ./...` to verify all unit and integration tests pass with >= 85% statement coverage across all packages in `internal/*`.
