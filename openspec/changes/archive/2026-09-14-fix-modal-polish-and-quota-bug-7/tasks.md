# Tasks: Modal Polish, State Synchronization & Quota Refresh (BUG-7)

## 1. Automated Tests (RED State) - [TEST-AGENT]

- [x] 1.1 Extend `TestWebFrontendConfirmDeleteModalStructure` in `services/storage-service/internal/handler/web_test.go` to assert `role="dialog"`, `aria-modal="true"`, `aria-labelledby="confirm-delete-title"`, `confirm-delete-target-name`, and `confirm-delete-folder-warning`.
- [x] 1.2 Extend `TestWebFrontendDeleteButtonStyles` in `services/storage-service/internal/handler/web_test.go` to verify `.confirm-delete-folder-warning` class presence in `css/modals.css`.
- [x] 1.3 Update `TestWebFrontendDeleteLogicJS` in `services/storage-service/internal/handler/web_test.go` to assert `textContent`, `cancelBtn.focus()`, `isDeleting`, and actual invocations of `deleteFile` and `deleteFolder`.
- [x] 1.4 Add test in `services/storage-service/internal/handler/web_test.go` asserting that `loadWorkspaceData` re-fetches user profile via `checkAuth` to synchronize quota, and that `Escape` key handler dismisses active modals.
- [x] 1.5 Run `go test ./...` in `services/storage-service` and verify new tests fail as expected (RED state).

## 2. Implementation (GREEN State) - [CODE-AGENT]

- [x] 2.1 Update `services/web-frontend/src/index.html` with ARIA dialog attributes (`role="dialog"`, `aria-modal="true"`, `aria-labelledby`), explicit `type="button"` attributes, `#confirm-delete-target-name`, and `#confirm-delete-folder-warning`.
- [x] 2.2 Add `.confirm-delete-folder-warning` class in `services/web-frontend/src/css/modals.css` and verify styling.
- [x] 2.3 Update `services/web-frontend/src/js/modals.js`: implement `closeTopModal()`, safe focus on Cancel upon open, `textContent` assignment, `isDeleting` in-flight lock, and relocate `handleConfirmDelete()` into `modals.js`.
- [x] 2.4 Update `services/web-frontend/src/js/app.js`: remove `handleConfirmDelete()`, connect `Escape` key listener to `closeTopModal()`, and re-fetch user profile in `loadWorkspaceData()` via `Promise.all([checkAuth(), loadFiles(), loadFolders()])`.
- [x] 2.5 Run `go test ./...` in `services/storage-service`, verify all tests pass (GREEN state), and verify that `TestMaxFileLineCount` passes with `app.js` well under 450 lines.
