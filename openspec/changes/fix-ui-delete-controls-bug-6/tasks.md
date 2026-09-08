# Tasks: UI Delete Controls for Files and Folders (BUG-6)

## 1. Test Suite (RED Phase - [TEST-AGENT])

- [x] 1.1 Add HTML structure tests in `internal/handler/web_test.go` verifying presence of `#modal-confirm-delete`, `#confirm-delete-title`, `#confirm-delete-msg`, `#confirm-delete-cancel`, and `#confirm-delete-btn` in `index.html`.
- [x] 1.2 Add CSS token tests in `internal/handler/web_test.go` verifying presence of `#modal-confirm-delete`, `.btn-danger`, and `.btn-icon-danger` in `css/modals.css`.
- [x] 1.3 Add JS logic tests in `internal/handler/web_test.go` verifying presence of `openConfirmDeleteModal`, `closeConfirmDeleteModal`, and delete button event bindings in frontend scripts.
- [x] 1.4 Verify all new tests fail in RED state (`go test -v ./internal/handler/... -run TestWebFrontend`) prior to implementation.


## 2. HTML & CSS Implementation ([CODE-AGENT])

- [x] 2.1 Add `#modal-confirm-delete` dialog markup to `services/web-frontend/src/index.html` with title, message text container, cancel button, and confirm delete button.
- [x] 2.2 Add modal backdrop, dialog styles, `.btn-danger`, and `.btn-icon-danger` to `services/web-frontend/src/css/modals.css` keeping the file well under 450 lines.

## 3. UI Rendering & Client Logic ([CODE-AGENT])

- [x] 3.1 Update `renderGrid()` in `services/web-frontend/src/js/ui.js` to render `.btn-action-delete` with trash SVG on both file and folder cards inside `.card-actions`.
- [x] 3.2 Update `renderList()` in `services/web-frontend/src/js/ui.js` to render `.btn-action-delete` in the Actions column for both files and folders, replacing empty dashes for folders.
- [x] 3.3 Add `openConfirmDeleteModal(id, type, name)` and `closeConfirmDeleteModal()` in `services/web-frontend/src/js/modals.js` with customized text and cascading folder warnings.
- [x] 3.4 Wire event listeners in `services/web-frontend/src/js/app.js` to invoke modal opening on delete button clicks (with `e.stopPropagation()`), handle confirm deletion via `window.api.deleteFile`/`window.api.deleteFolder`, display toasts, and refresh workspace state via `loadWorkspaceData()`.

## 4. Verification & Quality Gates ([CODE-AGENT] / [AUDIT-AGENT])

- [x] 4.1 Run `go test -v -cover ./...` in `services/storage-service` to verify all tests pass (GREEN state) with >= 85% statement coverage.
- [x] 4.2 Run `go test -v ./cmd -run TestMaxFileLineCount` to verify no modified or existing repository file exceeds the 450-line ceiling.
