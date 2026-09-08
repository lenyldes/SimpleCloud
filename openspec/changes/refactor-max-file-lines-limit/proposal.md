## Why

Large monolithic source files and oversized test suites degrade developer and AI agent productivity, exceed context inspection windows (such as `view_file` tool limits), increase token consumption, and make file modifications error-prone. By enforcing a universal hard ceiling of **<= 450 lines** across all readable text files in the repository, SimpleCloud ensures strict modularity, high cohesion, Single Responsibility Principle (SRP), and clean AI-assisted maintenance.

## What Changes

- **Universal File Line Limit Policy:** Formalize the architectural rule that ANY readable text file across the repository must not exceed 450 lines of code/text, by fundamental principle rather than a brittle extension whitelist.
- **Double-Check Binary Exclusion Logic:** Exclude strictly non-text binary files (fonts, images, compiled binaries) via known binary extensions and Git-style 4KB null-byte (`0x00`) / UTF-8 sniffing, plus `.git/` metadata.
- **Automated Architecture Test (`TestMaxFileLineCount`):** Add a deterministic Go test in `services/storage-service/cmd/architecture_test.go` that inspects all git-tracked files and asserts `lineCount <= 450` on all text files, failing fast in CI/CD with informative line counts and paths.
- **Backend File Handler Decomposition:** Decompose `services/storage-service/internal/handler/file.go` (534 lines) into `file_upload.go`, `file_download.go`, `file_list.go`, `file_delete.go`, and core `file.go` (structs and constructor) without altering public signatures or HTTP contracts.
- **Backend Test Suite Modularization:**
  - Decompose `services/storage-service/internal/handler/file_test.go` (1043 lines) into `file_upload_test.go`, `file_download_test.go`, `file_list_test.go`, and `file_delete_test.go`.
  - Decompose `services/storage-service/internal/handler/folder_test.go` (629 lines) into `folder_crud_test.go`, `folder_hierarchy_test.go`, and `folder_delete_test.go`.
  - Decompose `services/storage-service/internal/auth/handler_test.go` (479 lines) into `handler_auth_test.go` and `handler_session_test.go`.
- **Frontend JavaScript Modularization:** Modularize `services/web-frontend/src/app.js` (849 lines) into cohesive zero-build modules (`api.js`, `auth.js`, `ui.js`, `modals.js`, `app.js`) under `services/web-frontend/src/js/`, each <= 450 lines, preserving 100% of existing UI/UX interactions, CSP compliance, and zero build step.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `project-scaffold`: Adds requirement for repository-wide universal <= 450 lines file limit architectural guard and automated CI/CD test enforcement.
- `web-frontend`: Adds requirement for modular JavaScript architecture decomposition into cohesive script modules under 450 lines without build steps.

## Impact

- **Affected Code:** `services/storage-service/cmd/architecture_test.go`, `services/storage-service/internal/handler/*.go`, `services/storage-service/internal/auth/handler_test.go`, `services/web-frontend/src/index.html`, and `services/web-frontend/src/js/*.js` (replacing monolithic `app.js`).
- **APIs & Contracts:** Zero breaking changes. All HTTP endpoints, request/response formats, database models, and frontend UI behaviors remain 100% backward-compatible.
- **Dependencies & Tooling:** Zero external dependencies added. Standard Go library (`os`, `bytes`, `path/filepath`, `unicode/utf8`) used for architecture tests.
- **CI/CD:** `go test -v -cover ./...` in `.github/workflows/ci.yml` will automatically execute `TestMaxFileLineCount` on every push.
