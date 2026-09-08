## Why

The frontend stylesheet `services/web-frontend/src/styles.css` has grown to 939 lines. While an exhaustive audit revealed 0% dead code (all 73 classes and 10 IDs are in active use), monolithic CSS files exceeding 800 lines exceed agent tool viewing thresholds (`view_file` 800-line chunking) and increase cognitive and diff complexity. With Phase 7 UI deletion controls (BUG-6) and Phase 8 media features on the horizon, modularizing the stylesheet into 4 cohesive files within the agent sweet spot (80–350 lines) with a dedicated index map ensures maintainability and eliminates repetitive multi-file scanning for future tasks.

## What Changes

- **Decompose `styles.css` into 4 logical modules** under `services/web-frontend/src/css/`:
  - `base.css` (~85 lines): typography (`@font-face`), `:root` design tokens, universal reset (`*`), base `body` styles, accessibility utility (`.visually-hidden`).
  - `layout.css` (~250 lines): application container (`#app`), header bar (`#app-header`), brand block, search bar (`#search-input`), sidebar navigation (`#app-sidebar`), workspace container, breadcrumbs bar (`#breadcrumbs-bar`), toolbar & sort/view controls.
  - `components.css` (~350 lines): button primitives (`.btn`, `.btn-primary`, `.btn-icon`, etc.), user profile dropdown menu, storage quota container & progress bar, grid card layout (`.file-grid`), list table layout (`.file-list`), empty state indicator, toast notifications (`.toast`).
  - `modals.css` (~250 lines): modal overlay backdrop (`.modal-backdrop`), dialog layout (`.modal-dialog`), authentication modal (`#modal-auth`), new folder modal, media lightboxes (image lightbox, video player, code viewer), drag-and-drop overlay (`#dropzone-overlay`), form control groups.
- **Update `index.html`**: Replace the single `<link rel="stylesheet" href="styles.css">` with four parallel `<link rel="stylesheet">` tags referencing the modular files in proper cascade order.
- **Create In-Place Agent Index (`src/css/README.md`)**: A fast navigation cheat sheet mapping every UI component, class, and ID to its responsible CSS file so agents never need to scan across multiple files.
- **Update Architecture Guide (`docs/DESIGN_SPEC.md`)**: Document the modular CSS architecture and file responsibilities in the project design specification.
- **Preserve 100% Visual and Selector Parity**: All 73 classes, 10 IDs, custom properties, responsive properties, and visual behavior remain completely identical without regressions.

## Capabilities

### Modified Capabilities

- `web-frontend`: Adds modular CSS architecture and stylesheet delivery requirement, specifying file boundaries and in-place navigation index while maintaining 100% visual parity.

## Impact

- **Frontend Assets**: Deletes monolithic `services/web-frontend/src/styles.css` and creates `services/web-frontend/src/css/{base,layout,components,modals}.css`.
- **HTML Markup**: Modifies `services/web-frontend/src/index.html` stylesheet links.
- **Documentation**: Creates `services/web-frontend/src/css/README.md` and updates `docs/DESIGN_SPEC.md`.
- **Zero API/Backend Impact**: No changes to Go backend, database schema, Nginx routing, or Docker configurations.
