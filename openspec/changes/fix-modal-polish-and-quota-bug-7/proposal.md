# Proposal: Modal Polish, State Synchronization & Quota Refresh (BUG-7)

## Why

In manual testing (recorded as BUG-7 in `BUGS.md`), several UX and safety issues were discovered in the modal dialogs and workspace state management:
1. Modal dialogs ignore the `Escape` key, trapping users unless they click specific buttons.
2. In the delete confirmation dialog, closing controls remain active while an asynchronous deletion request is in flight, creating race conditions between user intent and server-side state.
3. Dialog opening lacks safe focus management, leaving keyboard focus on underlying table or grid action buttons.
4. Delete target names are injected via `innerHTML`, which introduces unnecessary XSS attack surface compared to DOM text nodes.
5. In-flight warning styling is hardcoded via inline style attributes instead of CSS classes.
6. Delete modal lacks accessible dialog ARIA attributes (`role="dialog"`, `aria-modal="true"`).
7. The sidebar quota progress bar (`used_bytes`) does not reflect uploads or deletions without a full page refresh (F5) because `state.user` is only fetched on initial load.
8. Frontend orchestrator `app.js` is at 443 lines, nearing the strict universal limit of <= 450 lines.

Resolving these issues ensures a robust, accessible, race-free, and safe modal interaction experience while keeping all files well within architectural line limits.

## What Changes

- **Centralized Escape Key Dismissal**: Pressing `Escape` closes the currently active top modal dialog (delete confirmation, new folder, image lightbox, text viewer, video player) and closes the user profile dropdown. The authentication modal is non-dismissible.
- **Race Condition Guard During Deletion**: While an asynchronous delete request is in flight (`isDeleting = true`), the delete button shows loading state, the Cancel button, close button, overlay click, and `Escape` key dismissal are disabled.
- **Safe Keyboard Focus on Confirmation**: Opening the confirm delete modal automatically transfers focus to the safe Cancel button.
- **XSS-Safe Target Name & Warning Rendering**: Replace `innerHTML` with a dedicated `#confirm-delete-target-name` element updated via `.textContent`, and toggle a structured `#confirm-delete-folder-warning` element using CSS class visibility.
- **CSS Class Extraction**: Move inline styling for folder deletion warning into `.confirm-delete-folder-warning` in `services/web-frontend/src/css/modals.css`.
- **Accessibility & ARIA Attributes**: Add `role="dialog"`, `aria-modal="true"`, `aria-labelledby="confirm-delete-title"`, and explicit `type="button"` attributes in `index.html`.
- **Dynamic Quota Synchronization**: In `loadWorkspaceData()`, query user profile via `checkAuth()` in parallel with file and folder listings (`Promise.all([checkAuth(), loadFiles(), loadFolders()])`) to refresh `state.user.used_bytes` and update the sidebar quota display after uploads and deletions.
- **Modular Deletion Logic Offloading**: Move `handleConfirmDelete()` and delete modal state from `app.js` to `modals.js`, reducing `app.js` from 443 to ~400 lines and maintaining the universal <= 450 lines limit.
- **Test Assertions Hardening**: Update Go frontend integration tests in `web_test.go` to assert invocation of `deleteFile` and `deleteFolder`, ARIA attributes, and warning styling.

## Capabilities

### Modified Capabilities

- `web-frontend`: Enhance modal dialog handling with Escape key support, deletion network request locks, Cancel button focus, textContent XSS protection, ARIA dialog accessibility, CSS warning classes, and automatic post-mutation used_bytes quota synchronization.

## Impact

- `services/web-frontend/src/index.html`: ARIA attributes, explicit button types, dedicated target name and warning DOM nodes.
- `services/web-frontend/src/css/modals.css`: New `.confirm-delete-folder-warning` class.
- `services/web-frontend/src/js/modals.js`: Relocate `handleConfirmDelete`, add `isDeleting` lock, focus Cancel on open, use `textContent` for target name, and provide centralized modal dismissal.
- `services/web-frontend/src/js/app.js`: Remove `handleConfirmDelete`, connect Escape listener to modal dismissal, refresh `checkAuth()` in `loadWorkspaceData()`, reducing line count well below 450 lines.
- `services/storage-service/internal/handler/web_test.go`: Assert actual API calls in delete handlers, ARIA modal attributes, and CSS class existence.
