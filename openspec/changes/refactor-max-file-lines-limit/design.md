## Context

SimpleCloud enforces strict engineering quality standards. Currently, 5 files exceed the 450-line limit:
- `services/storage-service/internal/handler/file_test.go` (1043 lines)
- `services/web-frontend/src/app.js` (849 lines)
- `services/storage-service/internal/handler/folder_test.go` (629 lines)
- `services/storage-service/internal/handler/file.go` (534 lines)
- `services/storage-service/internal/auth/handler_test.go` (479 lines)

See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Implement `TestMaxFileLineCount` in `cmd/architecture_test.go` using a double-check binary exclusion principle.
- Decompose all 5 violating files so every file in the repository is strictly <= 450 lines.
- Preserve 100% feature parity, API contracts, and keep statement coverage >= 85% in `internal/*`.
- Maintain zero-build simplicity for the web frontend.

**Non-Goals:**
- Introducing Node.js, Webpack, Vite, or npm build pipelines to the frontend.
- Refactoring backend business logic or changing database schemas.
- Modifying public API routes or JSON responses.

## Decisions

### Decision 1: Architecture Guard Implementation in Go Test Suite
- **Choice:** Standard Go test `TestMaxFileLineCount` in `services/storage-service/cmd/architecture_test.go`.
- **Rationale:** Automatically runs during `go test -v -cover ./...` both locally and in the GitHub Actions CI pipeline without needing separate external linter tools or shell scripts.
- **Alternatives Considered:** Standalone shell script or git pre-commit hook. Rejected because CI/CD test job is the definitive quality gate and developers run `go test` standardly.

### Decision 2: Inversion / Double-Check Sniffing over Extension Whitelist
- **Choice:** Test inspects all repository files discovered via `git ls-files` (with fallback to `filepath.WalkDir`). Exclusions are determined by:
  1. Known binary extensions (`.woff`, `.woff2`, `.png`, `.jpg`, `.jpeg`, `.gif`, `.ico`, `.webp`, `.pdf`, `.zip`, `.tar`, `.gz`) and `.git/`.
  2. Content sniffing of the first 4096 bytes for null bytes (`0x00`) or non-UTF-8 sequences.
- **Rationale:** Prevents artificial loopholes (e.g. creating huge `.txxx` or `.conf` files). Any readable text file is governed by the 450-line ceiling.
- **Alternatives Considered:** Rigid whitelist of `.go`, `.js`, `.css`. Rejected because agents or contributors could introduce oversized files under unlisted extensions.

### Decision 3: Go File Handler Package-Internal Decomposition
- **Choice:** Split `file.go` into cohesive files sharing `package handler`:
  - `file.go`: `FileHandler` struct, constructor `NewFileHandler`, common helpers.
  - `file_upload.go`: `UploadFile` multipart handler, quota check, disk write, DB insert.
  - `file_download.go`: `DownloadFile` streaming handler, Content-Disposition RFC 5987 formatting.
  - `file_list.go`: `ListFiles` query and sorting.
  - `file_delete.go`: `DeleteFile` deletion logic and quota rollback.
- **Rationale:** In Go, all files in the same directory share package scope. Private methods and fields remain accessible without exposing public interfaces or modifying routing.
- **Alternatives Considered:** Creating subpackages (e.g. `handler/upload`). Rejected as overengineering for an HTTP handler layer.

### Decision 4: Backend Test File Decomposition
- **Choice:** Split large test suites into domain-specific test files:
  - `file_test.go` -> `file_upload_test.go`, `file_download_test.go`, `file_list_test.go`, `file_delete_test.go`.
  - `folder_test.go` -> `folder_crud_test.go`, `folder_hierarchy_test.go`, `folder_delete_test.go`.
  - `auth/handler_test.go` -> `handler_auth_test.go`, `handler_session_test.go`.
- **Rationale:** Keeps each test file between 150 and 300 lines, focused on specific endpoints and scenarios.

### Decision 5: Frontend JavaScript Zero-Build Modularization
- **Choice:** Modularize `app.js` into cohesive scripts under `services/web-frontend/src/js/`:
  - `js/api.js`: Network fetch helpers, 401 interceptor, and backend API endpoints.
  - `js/auth.js`: User session check, auth modal display/submission, profile dropdown, logout.
  - `js/ui.js`: DOM rendering for file grid/list, breadcrumbs bar, quota indicator, and toasts.
  - `js/modals.js`: Dialog handlers for lightbox preview, text/code viewer, video player, and new folder modal.
  - `js/app.js`: Global state, DOM element caching, event listener setup, and `init()`.
- Scripts are included in `index.html` in dependency order with cache-busting query strings (`?v=1.2.0`).
- **Rationale:** Preserves zero-build architecture while cutting file sizes from 850 lines to ~150-250 lines per module.
- **Alternatives Considered:** ES Modules (`type="module"`). Rejected because standard script tags avoid CORS friction when opening local files and maintain consistent global event binding without bundling.

## Risks / Trade-offs

- **[Risk] Script dependency order in `index.html` causes undefined reference errors.**  
  → **Mitigation:** Strict dependency order: `api.js` -> `auth.js` -> `ui.js` -> `modals.js` -> `app.js`. Functions encapsulated in state/namespace objects, initialized after `DOMContentLoaded`.
- **[Risk] Coverage drops below 85% after splitting test files.**  
  → **Mitigation:** Test code is only moved, not deleted. `go test -cover ./...` will verify that total statement coverage remains >= 85%.
- **[Risk] Components CSS approaching 450 lines (currently 445).**  
  → **Mitigation:** `components.css` will be monitored during audit to ensure it remains strictly <= 450 lines.
