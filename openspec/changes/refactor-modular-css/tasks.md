## 1. Automated Parity & Structure Tests (Test Agent)

- [ ] 1.1 Create automated CSS parity and modularity test (`services/storage-service/internal/handler/css_parity_test.go` or a test verification suite) that checks:
  - `services/web-frontend/src/css/` directory exists and contains `base.css`, `layout.css`, `components.css`, and `modals.css`.
  - Each modular CSS file is within the target threshold (<= 450 lines).
  - All 73 CSS classes and 10 IDs from the original monolithic stylesheet are preserved without omission.
  - `index.html` links all 4 modular stylesheets in cascade order (`base.css` -> `layout.css` -> `components.css` -> `modals.css`) and no longer references `styles.css`.
  - `services/web-frontend/src/css/README.md` exists and contains the component-to-file lookup table.
  - Verify test fails in RED state prior to implementation.

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
