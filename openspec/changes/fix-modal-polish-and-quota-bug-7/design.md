# Design: Modal Polish, State Synchronization & Quota Refresh (BUG-7)

## Context

The SimpleCloud web UI is built using Vanilla JS modules (`app.js`, `modals.js`, `ui.js`, `auth.js`, `api.js`) and modular CSS files (`base.css`, `layout.css`, `components.css`, `modals.css`).
As identified in `BUGS.md` (BUG-7), modal interactions suffer from lack of Escape key dismissal, race conditions when closing dialogs during deletion requests, missing safe focus on open, XSS risk via `innerHTML` concatenation, inline styles, missing ARIA markup, and stale sidebar quota (`used_bytes`).
Crucially, `app.js` currently stands at 443 lines, with an architectural limit enforced by CI (`TestMaxFileLineCount`) of <= 450 lines.

## Goals / Non-Goals

**Goals:**
- Provide a clean, robust, and accessible modal interaction lifecycle across all dialogs.
- Prevent all asynchronous delete race conditions via an explicit in-flight state lock (`isDeleting`).
- Guarantee safe DOM manipulation (`textContent`) without `innerHTML` attack vectors.
- Eliminate all inline styles by introducing `.confirm-delete-folder-warning` in `modals.css`.
- Keep `app.js` and all other files well below 450 lines by offloading delete execution logic to `modals.js`.
- Keep sidebar quota dynamically synchronized with server-side `used_bytes` after every mutation.

**Non-Goals:**
- Redesigning the full router (`router.js` extraction is reserved for BUG-11).
- Changing backend deletion logic or database schema (already verified in Phase 7 Block 2).
- Making the authentication modal dismissible (unauthorized users must remain gated).

## Decisions

### 1. Relocate `handleConfirmDelete` from `app.js` to `modals.js`
- **Rationale**: `modals.js` currently contains `openConfirmDeleteModal`, `closeConfirmDeleteModal`, and `getPendingDeleteItem`, but the execution handler `handleConfirmDelete` was placed in `app.js`. `app.js` is at 443 lines (7 lines below the 450 limit), whereas `modals.js` is at 236 lines. Moving `handleConfirmDelete` to `modals.js` unifies all deletion dialog lifecycle logic in one module and frees ~40 lines in `app.js`, bringing it to ~400 lines.
- **Alternatives Considered**:
  - *Keep in `app.js`*: Would immediately violate the 450 lines ceiling once Escape and locking logic are added.
  - *Extract new file `delete_modal.js`*: Unnecessary fragmentation for a 40-line handler.

### 2. Centralized Modal Dismissal (`closeTopModal` / `closeActiveModals`)
- **Rationale**: When `Escape` is pressed, the application should dismiss the currently visible top-level overlay. In `modals.js`, expose a helper function `closeTopModal()` that checks open modals:
  - If `#modal-confirm-delete` is open: if `isDeleting` is true, ignore Escape; otherwise call `closeConfirmDeleteModal()`.
  - If `#modal-new-folder` is open: call `closeNewFolderModal()`.
  - If `#modal-lightbox` is open: call `closeLightbox()`.
  - If `#modal-text` is open: call `closeTextViewer()`.
  - If `#modal-video` is open: call `closeVideoPlayer()`.
  Returns `true` if a modal was closed, allowing `app.js` keydown listener to close the profile dropdown only when appropriate.
- **Alternatives Considered**:
  - *Registering individual keydown listeners inside each open/close function*: Prone to duplicate listeners and memory leaks.

### 3. In-Flight Deletion Lock (`isDeleting`)
- **Rationale**: Introduce a module-scoped boolean `isDeleting` in `modals.js`. When `handleConfirmDelete()` starts:
  - `isDeleting = true`
  - Disable `#confirm-delete-btn`, `#confirm-delete-cancel`, and `#confirm-delete-close`.
  - Update `#confirm-delete-btn` text to `'Deleting...'`.
  - In `closeConfirmDeleteModal()`, backdrop click listener, and `closeTopModal()`, reject closing if `isDeleting === true`.
  - In `finally`, restore `isDeleting = false`, re-enable buttons, and reset text.
- **Alternatives Considered**:
  - *Only disabling the delete button*: Leaves Cancel and backdrop clicks active, allowing users to close the UI while the HTTP DELETE request is already in flight.

### 4. DOM Nodes for Deletion Target Name & Warning (`textContent`)
- **Rationale**: In `index.html`, declare:
  ```html
  <div id="confirm-delete-msg" class="confirm-delete-msg">
    Are you sure you want to delete <strong id="confirm-delete-target-name"></strong>?
    <div id="confirm-delete-folder-warning" class="confirm-delete-folder-warning hidden">
      Warning: This will permanently delete the folder and all its contents.
    </div>
  </div>
  ```
  In `openConfirmDeleteModal`:
  - `targetNameEl.textContent = name;` (pure text node, immune to XSS injection).
  - Toggle `#confirm-delete-folder-warning` via `classList.toggle('hidden', type !== 'folder')`.
  - Focus `#confirm-delete-cancel` immediately (`cancelBtn.focus()`).

### 5. Asynchronous Quota Refresh via `checkAuth()` in `loadWorkspaceData()`
- **Rationale**: `loadWorkspaceData()` currently executes `Promise.all([loadFiles(), loadFolders()])`. After mutations (upload or delete), `loadWorkspaceData()` is invoked, but `state.user.used_bytes` remains stale. By expanding the parallel batch:
  ```javascript
  await Promise.all([
    typeof checkAuth === 'function' ? checkAuth() : Promise.resolve(),
    loadFiles(),
    loadFolders()
  ]);
  ```
  The user profile and `used_bytes` are refreshed concurrently without increasing sequential latency, and `updateQuotaDisplay()` immediately renders the accurate server quota.

## Risks / Trade-offs

- **[Risk: `app.js` exceeding 450 lines limit]** → *Mitigation*: Offloading `handleConfirmDelete` to `modals.js` reduces `app.js` to ~400 lines, ensuring plenty of margin for future maintenance.
- **[Risk: Latency from extra `GET /api/v1/auth/me` call]** → *Mitigation*: Executed in parallel with file and folder requests in `Promise.all()`, adding 0ms sequential wall time.
- **[Risk: Focus lost after modal close]** → *Mitigation*: Returning focus on modal close is standard, but keeping focus on Cancel upon opening prevents accidental destructive confirmation.
