## 1. Automated Parity & Structure Tests (Test Agent)

- [x] 1.1 Create permanent dynamic frontend integrity test (`services/storage-service/internal/handler/frontend_assets_test.go`) that acts as a continuous CI linter:
  - **Dynamic Class Presence**: Parses `index.html` and `app.js` to extract all referenced CSS classes, and asserts that 100% of these classes are defined in `services/web-frontend/src/css/*.css` (dynamic validation that scales automatically as new features/classes are added).
  - **File Modularity & Size Guard**: Asserts that `services/web-frontend/src/css/` contains `base.css`, `layout.css`, `components.css`, `modals.css`, and that no CSS module exceeds the 450-line agent limit.
  - **Link Integrity**: Asserts that all CSS files in `src/css/` are referenced via `<link rel="stylesheet">` in `index.html` in correct cascade order, and that obsolete `styles.css` is not referenced.
  - **Documentation Presence**: Asserts that `services/web-frontend/src/css/README.md` exists and contains the component lookup table.
  - Verify test executes and fails in RED state prior to implementation.

## 2. Modular CSS Decomposition & Markup Updates (Code Agent)

- [ ] 2.1 Create `services/web-frontend/src/css/` and extract modular stylesheets:
  - `base.css`: `@font-face`, `:root` design tokens, global reset, `body`, `.visually-hidden`, `.hidden`.
  - `layout.css`: `#app`, `#app-header`, `.brand-*`, `.search-*`, `#search-input`, `#app-sidebar`, `.nav-*`, `.main-content`, `.toolbar-*`, `#breadcrumbs-bar`, `.view-toggle`, `.sort-select`.
  - `components.css`: `.btn*`, `.user-profile*`, `.avatar`, `.profile-dropdown*`, `#quota-container*`, `.file-grid`, `.grid-card*`, `.file-list*`, `.empty-state*`, `#toast-container`, `.toast*`.
  - `modals.css`: `.modal-backdrop`, `.modal-dialog`, `.modal-header/body/footer`, `#modal-auth*`, `#modal-new-folder`, `.lightbox-*`, `.code-pre`, `.video-player`, `.form-*`, `#dropzone-overlay*`.
- [ ] 2.2 Update `services/web-frontend/src/index.html` to reference all 4 modular stylesheets in strict cascade order with cache-busting query parameter `?v=1.2.0`, and remove `services/web-frontend/src/styles.css`.
- [ ] 2.3 Create `services/web-frontend/src/css/README.md` containing the in-place navigation guide, file responsibilities, and the Component-to-File lookup table.
- [ ] 2.4 Update `docs/DESIGN_SPEC.md` to document the modular CSS architecture and file breakdown.
- [ ] 2.5 Run test suite (`go test -v ./...` in `services/storage-service`) to verify GREEN state with 100% selector parity.
