## Context

See `proposal.md` for motivation and background.

The backend endpoint `POST /api/v1/auth/logout` is already fully implemented in `services/storage-service/internal/auth/handler.go` (`LogoutHandler`) and tested. It deletes the active session from PostgreSQL and expires the `session_id` cookie via `http.SetCookie` (`MaxAge: -1`, `Path: /`, `HttpOnly: true`).

On the frontend (`services/web-frontend`), `index.html` has `#user-profile` with only an avatar element (`#user-avatar`). The avatar has no dropdown structure, CSS styles for a dropdown are absent, and `app.js` contains no event handlers for user profile interaction or logout calls.

## Goals / Non-Goals

**Goals:**
- Provide a clean, modern user profile dropdown menu inside the header `#user-profile` container.
- Display the authenticated user's email in `#profile-dropdown-email` and a styled logout button `#btn-logout`.
- Implement toggle logic on avatar click with outside-click dismissal in `app.js`.
- Execute `POST /api/v1/auth/logout` in `handleLogout()`, clear client memory (`state.user`, `state.files`, `state.folders`, `state.currentFolderId`), clear workspace DOM, open `showAuthModal()`, and trigger a confirmation toast.
- Ensure fallback client cleanup if the network request fails.
- Expand Go test assertions in `frontend_integration_test.go` to enforce `#profile-dropdown`, `#btn-logout`, and logout API calls.

**Non-Goals:**
- Modifying backend Go code or database schemas (`storage-service` is already complete for logout).
- Profile editing features (name/email changing, avatar upload, password changes) — reserved for future phases.
- Multi-user account switching.

## Decisions

### 1. Dropdown Structure & Absolute Positioning
- **Choice**: Place `#profile-dropdown` directly inside `#user-profile`. Give `#user-profile` `position: relative` and `#profile-dropdown` `position: absolute; right: 0; top: calc(100% + 8px); z-index: 100`. Include a small SVG chevron indicator next to the avatar.
- **Rationale**: Self-contained component; natural viewport alignment on the right without dynamic JS coordinate calculations.
- **Alternatives considered**: Fixed positioning appended to `document.body` — rejected because it requires JS `getBoundingClientRect()` on scroll and resize.

### 2. Outside Click & Toggle Lifecycle
- **Choice**: Toggle class `.open` on `#profile-dropdown` when `#user-profile` is clicked. Add a global document click listener: if `!userProfile.contains(e.target)`, remove `.open`. Stop propagation or check target to prevent premature closing when clicking inside the dropdown.
- **Rationale**: Idiomatic Vanilla JS pattern with zero dependencies.
- **Alternatives considered**: Native `<dialog>` or `<details>` element — `<details>` has browser styling inconsistencies with flex containers and dismiss-on-outside-click behavior across browsers.

### 3. Graceful State Purge & 401 Handling
- **Choice**: In `handleLogout()`, send `POST /api/v1/auth/logout` with `credentials: 'include'`. Regardless of network success or error, perform client-side cleanup: reset `state.user = null`, `state.files = []`, `state.folders = []`, reset folder path, clear DOM workspace, hide the dropdown, call `showAuthModal()`, and show toast notification.
- **Rationale**: Prevents stale data leakage in DOM if network is flaky or server returns 500.

## Risks / Trade-offs

- **[Risk] Parent header overflow clipping dropdown** → Mitigation: Ensure header and container have `overflow: visible` and dropdown has `z-index: 100` above workspace elements.
- **[Risk] Duplicate logout requests while request is in flight** → Mitigation: Disable `#btn-logout` immediately on click while awaiting response.
- **[Risk] Stale email in dropdown when switching accounts** → Mitigation: Update `#profile-dropdown-email` whenever `state.user` is loaded (in `checkAuth()` and `handleLogin()`).
