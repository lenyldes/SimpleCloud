# Design: UI Delete Controls for Files and Folders (BUG-6)

## Context

SimpleCloud uses a decoupled frontend architecture (`services/web-frontend/src`) served by Nginx and communicating with Go `storage-service` over REST. The backend handlers (`DELETE /api/v1/files/:id` and `DELETE /api/v1/folders/:id`) and the frontend client methods (`window.api.deleteFile` and `window.api.deleteFolder` in `js/api.js`) already exist. What is missing is the user interface representation: buttons on cards/rows and a confirmation dialog.

The codebase strictly enforces a universal <= 450 lines limit on every text file. Currently `services/web-frontend/src/css/components.css` has 445 lines, and `services/storage-service/internal/handler/frontend_integration_test.go` has 447 lines. Architectural decisions must respect these ceilings.

## Goals / Non-Goals

**Goals:**
- Provide accessible delete buttons with trash icons on file and folder cards in Grid View and row action cells in List View.
- Implement an accessible, responsive confirmation modal (`#modal-confirm-delete`) in `index.html` with explicit cascade warnings for folders.
- Prevent unintended file opening or folder navigation when clicking delete controls via event isolation.
- Ensure workspace items and quota usage automatically update upon successful deletion.
- Keep all modified and existing files strictly within the 450-line ceiling.

**Non-Goals:**
- Modifying preview modals (Lightbox, Text viewer, Video player) with delete buttons at this stage.
- Implementing a "Trash" / "Recycle Bin" soft-delete recovery mechanism (backend performs immediate permanent deletion).
- Modifying backend Go delete handler logic or database schemas.

## Decisions

### 1. Custom Confirmation Modal vs Browser Native Dialog
- **Choice**: Custom modal dialog `#modal-confirm-delete` styled with SimpleCloud design system (`css/modals.css`).
- **Rationale**: Browser `window.confirm()` blocks JS execution, cannot be styled, and lacks visual hierarchy for dangerous cascading operations. The custom modal allows displaying the specific item name in bold and a clear red badge/text for folder cascading risks.
- **Alternatives Considered**: Native `confirm()` (poor UX, inconsistent across browsers).

### 2. Styles Placement in `modals.css`
- **Choice**: Place all `#modal-confirm-delete`, `.btn-danger`, and `.btn-icon-danger` styles in `services/web-frontend/src/css/modals.css`.
- **Rationale**: `components.css` is at 445 lines (5 lines below the 450 hard ceiling). Adding button styles there would risk violating `TestMaxFileLineCount`. `modals.css` has 186 lines, providing ample headroom.
- **Alternatives Considered**: Adding to `components.css` (violates 450 lines limit).

### 3. Test Placement in `web_test.go`
- **Choice**: Place Go regression tests for frontend delete assets in `services/storage-service/internal/handler/web_test.go`.
- **Rationale**: `frontend_integration_test.go` is at 447 lines (3 lines below 450 limit). `web_test.go` is at 167 lines and already contains structure and asset checks for auth and navigation modals.
- **Alternatives Considered**: Creating a new file `frontend_delete_test.go` (unnecessary file proliferation when `web_test.go` has 280 lines of spare headroom).

### 4. Event Isolation and Delegation
- **Choice**: In `js/ui.js`, wrap actions in `.card-actions` (grid) and `.list-actions` (list). Add `e.stopPropagation()` on delete button click handlers, and update grid/list row click listeners to ignore events originating from `.card-actions`, `.list-actions`, or `button`.
- **Rationale**: Prevents clicks on the delete button from bubbling up to the row/card click handler, which would otherwise trigger folder opening or file download.

### 5. Pending Deletion State in Client
- **Choice**: Store pending item metadata `{ id, type, name }` in module state in `js/modals.js` when `openConfirmDeleteModal(id, type, name)` is called.
- **Rationale**: Keeps `index.html` clean of heavy data attributes and allows the confirmation click handler to simply invoke `handleDeleteConfirm()`.

## Risks / Trade-offs

- **[Accidental deletion of folders with nhiều files]** → Mitigation: Explicit warning message in `#modal-confirm-delete` stating: "This will permanently delete the folder and all its contents." with an unmistakable `.btn-danger` button.
- **[450-line file limit regression in CI]** → Mitigation: Keep additions concise and place styles in `modals.css` and tests in `web_test.go`.
- **[Network failure during deletion]** → Mitigation: Catch HTTP errors from `window.api.delete*` and display an error toast notification without closing the modal or desynchronizing UI state.
