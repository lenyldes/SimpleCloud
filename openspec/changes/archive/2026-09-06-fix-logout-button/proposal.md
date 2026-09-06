## Why

Currently, users who authenticate to SimpleCloud cannot log out from the web interface because the user profile avatar in the header is static and lacks an interactive dropdown menu or logout button (tracked in `BUGS.md` as BUG-1 and Checklist item 1.4). Users are forced to manually delete `session_id` cookies via browser DevTools to end a session. Adding a profile dropdown menu with a logout action provides essential session control, aligns the interface with modern cloud storage UX standards, and resolves BUG-1.

## What Changes

- **HTML Layout**: Extend `#user-profile` in `services/web-frontend/src/index.html` with a dropdown chevron and a profile dropdown menu container (`#profile-dropdown`), displaying the authenticated user's email and a logout button (`#btn-logout`).
- **Dropdown Styling**: Add CSS rules in `services/web-frontend/src/styles.css` for `.profile-dropdown`, including absolute positioning below the avatar, drop shadow, rounded corners, user email badge styling, hover states, and smooth open/close visibility transitions.
- **Interactive State & Outside Click**: In `services/web-frontend/src/app.js`, add event listeners to toggle the profile dropdown when clicking the user profile container, and close the dropdown when clicking anywhere outside of `#user-profile`.
- **Logout Action**: Implement `handleLogout()` in `services/web-frontend/src/app.js` that calls `POST /api/v1/auth/logout` with credentials, resets client-side state (`state.user = null`, `state.files = []`, `state.folders = []`), closes the dropdown, resets the file listing/workspace view, opens the auth modal (`showAuthModal()`), and presents a toast notification.
- **Frontend Integration Tests**: Expand Go assertions in `services/storage-service/internal/handler/frontend_integration_test.go` to verify the presence of `#btn-logout` and `#profile-dropdown` in `index.html`, dropdown CSS rules in `styles.css`, and logout endpoint calls in `app.js`.

## Capabilities

### Modified Capabilities

- `web-frontend`: Add requirement for interactive User Profile Dropdown and Logout action in the web interface.

## Impact

- Frontend static assets: `services/web-frontend/src/index.html`, `services/web-frontend/src/styles.css`, `services/web-frontend/src/app.js`.
- Go test suite: `services/storage-service/internal/handler/frontend_integration_test.go`.
- Bug tracking: Resolves BUG-1 in `BUGS.md` and fulfills Checklist item 1.4.
- Backend API: Uses the existing, tested `POST /api/v1/auth/logout` endpoint in `storage-service` without requiring backend API schema changes.
