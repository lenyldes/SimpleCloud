# Proposal: Fix Folder Navigation and Hierarchy Browsing (BUG-8 & BUG-9)

## Why

During manual user acceptance testing of Phase 7 (checklist items 3.3 and 3.4), critical defects were identified in directory browsing and hierarchy state management:
1. Creating nested subfolders does not display them in the workspace because the frontend API client ignores `parent_id` when listing folders, causing `loadWorkspaceData()` to overwrite local folder state with only root-level directories (BUG-8).
2. Reloading the page (F5) throws the user back to the root directory because the current folder state is kept purely in-memory with no URL route or hash persistence (BUG-8).
3. Drag-and-dropping a directory from the operating system triggers a network failure because the upload handler attempts to read the directory entry as a standard file part (BUG-8).
4. Navigating via breadcrumbs back to ancestor folders or the root leaves `state.files` populated with child folder files without triggering a network re-fetch, causing root files to falsely disappear from view (BUG-9).

Resolving these issues ensures reliable multi-level folder navigation, keeps UI state strictly synchronized with backend PostgreSQL records, preserves active directory view across page reloads without sacrificing tenant isolation (`WHERE user_id = $1`), and prevents unexpected file disappearing.

## What Changes

- **Frontend API Client (`api.js`)**: Update `apiListFolders(parentId = null)` to support passing `parent_id` query parameter (`/api/v1/folders?parent_id=...`) matching the backend contract.
- **Application Orchestrator (`app.js`)**:
  - Pass `state.currentFolderId` when calling `api.listFolders(...)` inside `loadFolders()`.
  - Introduce safe URL-hash routing (`#/folder/<uuid>`) to persist active folder across page refreshes (F5) and browser back/forward navigation.
  - Implement tenant isolation guards: if an invalid or unauthenticated/foreign folder UUID is present in the URL hash, automatically redirect to root (`#/`) and alert the user without leaking data.
  - Enhance `setupDragAndDrop` with folder drop detection: if a user drops a folder, halt upload and display a clear notification toast.
- **Workspace UI Navigation (`ui.js`)**:
  - Update `navigateToFolder` and `navigateToBreadcrumb` to trigger dynamic network re-fetching (`loadWorkspaceData`) upon directory changes, ensuring `state.files` and `state.folders` always reflect the target level.
  - Reconstruct breadcrumb hierarchy when opening deep folder URLs on page load.
- **Test Suite (`web_test.go`)**: Add automated regression tests validating `parent_id` parameter propagation, breadcrumb network reload invocation, URL-hash navigation, and folder drop safety.

## Capabilities

### Modified Capabilities
- `web-frontend`: Clarifies and expands requirements for folder listing with `parent_id`, reactive workspace reload on breadcrumb navigation, URL-hash route persistence with tenant isolation, and folder drag-and-drop validation.

## Impact

- **Affected Files**:
  - `services/web-frontend/src/js/api.js`
  - `services/web-frontend/src/js/app.js`
  - `services/web-frontend/src/js/ui.js`
  - `services/storage-service/internal/handler/web_test.go`
  - `BUGS.md` (marked in-progress / resolved upon testing)
- **API Parity**: 100% adherence to existing backend contracts (`GET /api/v1/folders?parent_id=...` and `GET /api/v1/files?folder_id=...`). No backend schema migrations required.
- **File Length Limits**: Universal <= 450 lines per file maintained.
