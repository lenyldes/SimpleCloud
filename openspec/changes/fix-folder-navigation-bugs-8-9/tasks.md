# Tasks: Fix Folder Navigation and Hierarchy Browsing (BUG-8 & BUG-9)

## 1. Automated Tests (RED State)

- [x] 1.1 Add backend unit/integration tests in `folder_hierarchy_test.go` verifying `GET /api/v1/folders?all=true` returns all user folders and excludes other users' folders
- [x] 1.2 Add frontend JavaScript integration tests in `web_nav_test.go` verifying that `apiListFolders` passes `parent_id`, `loadFolders` propagates active folder ID, and `navigateToFolder` / `navigateToBreadcrumb` trigger network re-fetching
- [x] 1.3 Add frontend tests in `web_nav_test.go` verifying URL-hash parsing (`#/folder/<id>`), F5 recovery, safe redirect to root on foreign or invalid UUIDs, and folder drop validation in `setupDragAndDrop`

## 2. Backend Folder Listing Enhancement

- [ ] 2.1 Update `ListHandler` in `services/storage-service/internal/handler/folder.go` to support `?all=true`, querying all folders for the authenticated user (`WHERE user_id = $1 ORDER BY created_at ASC`), while preserving existing `parent_id` behavior and verify tests pass

## 3. Frontend API Client & State Orchestrator

- [ ] 3.1 Update `apiListFolders(parentId = null)` in `services/web-frontend/src/js/api.js` to append `?parent_id=${encodeURIComponent(parentId)}` when `parentId` is provided, and add `apiListAllFolders()` supporting `?all=true`
- [ ] 3.2 Update `loadFolders()` in `services/web-frontend/src/js/app.js` to pass `state.currentFolderId` when querying child folders and keep `state.allFolders` populated for breadcrumb path resolution
- [ ] 3.3 Implement client URL-hash router in `services/web-frontend/src/js/app.js`: parse `#/folder/<id>` on `DOMContentLoaded` / `hashchange`, validate folder existence in user hierarchy, safely redirect to `#/` on foreign/invalid IDs, and synchronize `state.currentFolderId`
- [ ] 3.4 Update `setupDragAndDrop` in `services/web-frontend/src/js/app.js` to detect directory drops, prevent invalid multipart uploads, and display an informative notification toast

## 4. Frontend UI Navigation & Breadcrumbs

- [ ] 4.1 Refactor `navigateToFolder(folder)` in `services/web-frontend/src/js/ui.js` to update URL hash and invoke `loadWorkspaceData()` so that `state.files` and `state.folders` are cleanly refreshed from the server
- [ ] 4.2 Refactor `navigateToBreadcrumb(folderId)` in `services/web-frontend/src/js/ui.js` to update URL hash and invoke `loadWorkspaceData()`, restoring ancestor files and subfolders without state corruption
- [ ] 4.3 Update `renderBreadcrumbs()` in `services/web-frontend/src/js/ui.js` to dynamically construct the ancestor path using `state.allFolders` so breadcrumb chains remain accurate even on direct URL entry or F5 reload
- [ ] 4.4 Update `handleCreateFolder` in `services/web-frontend/src/js/modals.js` to re-fetch folder hierarchy and workspace data so newly created subfolders appear immediately in the active folder

## 5. Verification & Quality Gates

- [ ] 5.1 Run Go formatting (`gofmt -s -w .`) and verify all automated tests pass with >= 85% statement coverage (`go test -v -cover ./...`)
- [ ] 5.2 Verify repository-wide Universal <= 450 lines file ceiling (`go test -v -run TestMaxFileLineCount ./...`)
