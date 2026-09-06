## 1. Test Suite Extension (RED State - [TEST-AGENT])

- [x] 1.1 Add assertions in `frontend_integration_test.go` for `#profile-dropdown` and `#btn-logout` in `index.html`, and verify test execution fails (RED).
- [x] 1.2 Add assertions in `frontend_integration_test.go` for profile dropdown CSS rules in `styles.css` and `/api/v1/auth/logout` reference in `app.js`, and verify test execution fails (RED).

## 2. Web Frontend Layout and Styles (GREEN State - [CODE-AGENT])

- [ ] 2.1 Update `services/web-frontend/src/index.html` to expand `#user-profile` with a dropdown chevron and `#profile-dropdown` containing user email display and `#btn-logout`.
- [ ] 2.2 Add CSS rules in `services/web-frontend/src/styles.css` for `.profile-dropdown`, `.profile-dropdown.open`, profile header/email, and logout button hover states.

## 3. Web Frontend Interactivity and Session Termination (GREEN State - [CODE-AGENT])

- [ ] 3.1 Implement profile dropdown toggling and document click-outside dismissal in `services/web-frontend/src/app.js`.
- [ ] 3.2 Update profile email display in `#profile-dropdown` on authentication in `services/web-frontend/src/app.js`.
- [ ] 3.3 Implement `handleLogout()` in `services/web-frontend/src/app.js` with `POST /api/v1/auth/logout`, client state purge, auth modal display, and toast notification.

## 4. Verification and Quality Checks (GREEN State - [CODE-AGENT] & [AUDIT-AGENT])

- [ ] 4.1 Execute `go test -v -cover ./...` in `services/storage-service` to confirm all frontend integration and backend tests pass with >=85% statement coverage.
