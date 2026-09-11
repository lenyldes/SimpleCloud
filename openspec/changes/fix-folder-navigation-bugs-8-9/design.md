# Technical Design: Folder Navigation, Hierarchy Browsing & Route Persistence

## Context

See `proposal.md` for background and problem motivation.

Currently, the Go backend implements hierarchical folder storage via PostgreSQL (`folders` table with `id`, `user_id`, `parent_id`, `name`) and provides `GET /api/v1/folders?parent_id=...` and `GET /api/v1/files?folder_id=...`. However, the Vanilla JS frontend (`api.js`, `app.js`, `ui.js`) experiences two critical failures:
1. `api.js` ignores `parent_id` when fetching folders, and `app.js` calls it without `state.currentFolderId`. When a subfolder is created, `loadWorkspaceData()` overwrites the local folder cache with root-only folders, hiding the newly created subfolder (BUG-8).
2. `navigateToFolder` and `navigateToBreadcrumb` in `ui.js` mutate `state.currentFolderId` in-memory and re-render without triggering network re-fetching. Uploading a file in a subfolder overwrites `state.files` with subfolder files, and navigating back to root filters `state.files` locally, causing root files to disappear (BUG-9). Furthermore, page refresh (F5) resets the in-memory state to root because no URL routing exists.

## Goals / Non-Goals

**Goals:**
- **API & State Parity**: Ensure `api.js` and `app.js` pass `parent_id` / `folder_id` matching backend contracts so directory views and subfolder creation stay synchronized.
- **Reactive Navigation**: Ensure `navigateToFolder` and `navigateToBreadcrumb` trigger network re-fetching (`loadWorkspaceData`) so `state.files` and `state.folders` always accurately reflect the active level.
- **URL-Hash Routing with Tenant Isolation**: Maintain active folder state via `#/folder/<uuid>` across page reloads (F5) and browser back/forward buttons, strictly isolated to the authenticated user (`WHERE user_id = $1`). Nonexistent or foreign folder UUIDs must safely redirect to root (`#/`) with a warning toast.
- **OS Folder Drop Protection**: Detect directory drag-and-drop in `setupDragAndDrop`, preventing failed multipart uploads and alerting the user.
- **Architectural Constraints**: Preserve the Universal <= 450 lines limit across all modified files and maintain >= 85% statement test coverage.

**Non-Goals:**
- Recursive directory tree uploading (drag-and-drop folder upload from OS is rejected gracefully, not implemented as a recursive multi-level upload in this bug fix).
- Public link sharing or unauthenticated folder browsing.
- Database schema changes (existing PostgreSQL schema already supports nested `parent_id`).

## Decisions

### Decision 1: Client-Side URL-Hash Routing (`#/folder/<uuid>`)
- **Choice**: Use hash-based routing (`window.location.hash`) listening to `window.addEventListener('hashchange', ...)` and inspecting `window.location.hash` during `init()`.
- **Rationale**: SimpleCloud uses a decoupled Nginx reverse proxy serving static files for the frontend. HTML5 History API (`pushState` with path routes like `/folder/<uuid>`) would require custom Nginx fallback rewrites (`try_files $uri $uri/ /index.html;`) and could conflict with API proxies. Hash-based routing works 100% reliably out of the box in all browsers and reverse proxies.
- **Tenant Isolation Protocol**: When `hashchange` fires or `init()` reads `#/folder/<uuid>`, the frontend validates whether the folder exists in the user's folder hierarchy. If the folder is missing or unauthorized, the hash is reset to `#/`, `state.currentFolderId` is set to `null`, and a notification is displayed.

### Decision 2: Backend Folder Tree Query (`all=true`) & Breadcrumb Path Resolution
- **Choice**: Extend Go backend `ListHandler` in `folder.go` to support `GET /api/v1/folders?all=true`, returning all folders for the authenticated user (`WHERE user_id = $1 ORDER BY created_at ASC`).
- **Rationale**: 
  - To reconstruct breadcrumb trails (`All Files / Project / 2026 / Docs`) when a user opens a deep URL directly or presses F5, the frontend needs the chain of ancestor folder names and IDs.
  - Querying all folders for a single user is lightweight (<10 KB JSON) and securely scoped to `WHERE user_id = $1`.
  - The frontend can store `state.allFolders` to immediately reconstruct ancestor breadcrumbs from any node up to root.
  - Backward compatibility is strictly preserved: `GET /api/v1/folders` without parameters continues returning root folders (`parent_id IS NULL`), and `GET /api/v1/folders?parent_id=<id>` returns direct children.

### Decision 3: Reactive Network Data Loading on Navigation
- **Choice**: Refactor `navigateToFolder(folder)` and `navigateToBreadcrumb(folderId)` to set `window.location.hash` (which triggers route handler) or invoke `loadWorkspaceData()`.
- **Rationale**: Eliminates the stale in-memory cache desynchronization where `state.files` held subfolder files and failed root filters. Every directory transition now loads the exact files (`GET /api/v1/files?folder_id=...`) and folders (`GET /api/v1/folders?parent_id=...` or filtered from `state.allFolders`).

### Decision 4: Operating System Folder Drop Detection
- **Choice**: In `setupDragAndDrop`, inspect `e.dataTransfer.items` or `e.dataTransfer.files`. In modern browsers, `item.webkitGetAsEntry && item.webkitGetAsEntry().isDirectory` or `file.size === 0 && (file.type === '' && !file.name.includes('.'))` can detect folder drops.
- **Rationale**: Prevents sending broken multipart requests that return 400/500 errors to the backend. Shows an informative toast: `"Загрузка папок не поддерживается. Пожалуйста, создайте папку и загрузите файлы внутрь"`.

## Risks / Trade-offs

- **[Risk] User enters an invalid or foreign UUID in the URL hash** → **Mitigation**: The frontend verifies `targetFolder` against the user's authorized folders returned by the backend. If not found, it immediately clears the hash to `#/`, shows a toast `"Folder not found or access denied"`, and loads root.
- **[Risk] Rapid clicking on breadcrumbs causes race conditions in fetch requests** → **Mitigation**: Track an incrementing request sequence token in `app.js` so older, out-of-order fetch responses are ignored if a newer navigation has started.
- **[Risk] File line limit exceedance (>450 lines)** → **Mitigation**: `web_test.go` and `app.js` are close to 370 lines. We will decompose router logic into clean, concise functions and, if needed, split test suites (`web_nav_test.go`).
