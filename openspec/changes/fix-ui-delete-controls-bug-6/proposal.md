# Proposal: UI Delete Controls for Files and Folders (BUG-6)

## Why

While the SimpleCloud backend provides robust endpoints for deleting files (`DELETE /api/v1/files/:id`) and folders (`DELETE /api/v1/folders/:id`) with transactional quota adjustments and cascading filesystem cleanup, the web frontend lacks UI controls to trigger these operations. Users currently have no way to remove unwanted files or delete folders from the browser interface. Resolving BUG-6 restores parity between backend capabilities and user-facing frontend actions.

## What Changes

- **Add Delete Button to Grid View**: Render a delete action button (`.btn-action-delete`) inside `.card-actions` for both file cards and folder cards in `renderGrid()`.
- **Add Delete Button to List View**: Render a delete action button (`.btn-action-delete`) inside the `Actions` column for both files and folders in `renderList()`, replacing the empty dash placeholder (`-`) for folders.
- **Confirmation Dialog Modal**: Introduce a custom styled modal `#modal-confirm-delete` in `index.html` featuring the item name, an explicit warning for folder deletions (highlighting cascading deletion of nested files and subfolders), a `Cancel` button, and a destructive `Delete` action button (`.btn-danger`).
- **Modal & Danger Button Styles**: Add styles for `#modal-confirm-delete`, `.btn-danger`, and `.btn-icon-danger` in `css/modals.css` to respect the <= 450 lines limit on all stylesheet files.
- **Client Handlers & State Synchronization**: Wire modal opening and confirmation handlers in `js/modals.js` and `js/app.js` using `window.api.deleteFile()` and `window.api.deleteFolder()`. Include event propagation isolation (`e.stopPropagation()` and `.closest()`) to prevent accidental folder navigation or file opening on delete clicks. On deletion, show toast feedback and refresh workspace items and quota via `loadWorkspaceData()`.
- **Automated Frontend Regression Tests**: Extend `internal/handler/web_test.go` with verification tests for confirmation modal markup, CSS design classes, and JavaScript delete orchestration.

## Capabilities

### New Capabilities
*(None)*

### Modified Capabilities
- `web-frontend`: Add UI controls for deleting files and folders in grid and list layouts with custom modal confirmation and workspace state synchronization.

## Impact

- **Affected Code**:
  - `services/web-frontend/src/index.html` (modal markup)
  - `services/web-frontend/src/css/modals.css` (modal and danger button styles)
  - `services/web-frontend/src/js/ui.js` (rendering delete icons in grid and list views)
  - `services/web-frontend/src/js/modals.js` (modal open/close logic and delete confirmation)
  - `services/web-frontend/src/js/app.js` (event wiring and workspace refresh)
  - `services/storage-service/internal/handler/web_test.go` (Go tests for frontend assets parity)
- **APIs**: Consumes existing `DELETE /api/v1/files/:id` and `DELETE /api/v1/folders/:id` (no API changes required).
- **Dependencies**: Zero new dependencies; uses existing vanilla JS and modular CSS architecture.
