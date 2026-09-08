## Context

`services/web-frontend/src/styles.css` contains 939 lines of Vanilla CSS serving the entire SimpleCloud web UI. The frontend is a static web application served via Nginx without build tools (no Node.js/bundler). While the CSS is clean with 0% dead code, the file size is approaching 1000 lines, exceeding the 800-line tool threshold for AI agents and making maintenance cumbersome.

## Goals / Non-Goals

**Goals:**
- Decompose `styles.css` into 4 cohesive modular CSS files in `services/web-frontend/src/css/` (each 80–350 lines).
- Maintain 100% cascade order, selector specificity, and visual parity with zero visual regressions.
- Establish an in-place cheat sheet in `services/web-frontend/src/css/README.md` mapping UI components to their files for instant agent navigation.
- Update `docs/DESIGN_SPEC.md` with the new CSS architecture.
- Load stylesheets concurrently via `<link rel="stylesheet">` in `index.html`.

**Non-Goals:**
- Introducing any bundler, compiler, or npm dependencies (e.g. Vite, PostCSS, Sass).
- Modifying design tokens, color values, typography, or DOM class names.
- Touching backend Go services, database schemas, or Docker Compose networking.

## Decisions

### 1. Parallel `<link>` Elements vs `@import`

- **Decision**: Load modules via 4 `<link rel="stylesheet">` tags in `index.html`.
- **Rationale**: CSS `@import` introduces sequential HTTP request waterfalls that delay first meaningful paint. `<link>` tags allow browsers to request all stylesheets in parallel over HTTP/1.1 Keep-Alive or HTTP/2 without latency penalty.
- **Alternatives considered**:
  - Single `styles.css` entry with 4 `@import` statements: Rejected due to request waterfall overhead.
  - Adding a build step (Vite/esbuild) to bundle CSS: Rejected to keep the project ultra-lightweight and zero-build.

### 2. Module Boundaries & File Taxonomy

| File | Size (est.) | Primary Content & Selectors |
| :--- | :--- | :--- |
| `base.css` | ~85 lines | `@font-face` (Inter), `:root` design tokens, `*` box-sizing/reset, `body`, `.visually-hidden`, `.hidden`. |
| `layout.css` | ~250 lines | `#app`, `#app-header`, `.brand-*`, `.search-*`, `#search-input`, `#app-sidebar`, `.nav-menu`, `.nav-item`, `.main-content`, `.toolbar-*`, `#breadcrumbs-bar`, `.view-toggle`, `.sort-select`. |
| `components.css` | ~350 lines | `.btn*` (primary, secondary, outline, icon), `.user-profile*`, `.avatar`, `.profile-dropdown*`, `#quota-container*`, `.file-grid`, `.grid-card*`, `.file-list*`, `.empty-state*`, `#toast-container`, `.toast*`. |
| `modals.css` | ~250 lines | `.modal-backdrop`, `.modal-dialog`, `.modal-header/body/footer/title`, `#modal-auth*`, `#modal-new-folder`, `.lightbox-*`, `.code-pre`, `.video-player`, `.form-*`, `#dropzone-overlay*`. |

### 3. Cascade Loading Sequence

The files MUST be linked in `index.html` in the following strict order:
1. `css/base.css?v=1.2.0` (variables, reset, typography)
2. `css/layout.css?v=1.2.0` (scaffolding containers)
3. `css/components.css?v=1.2.0` (UI controls & workspace items)
4. `css/modals.css?v=1.2.0` (overlays & interactive dialogs)

This preserves identical specificity and override behavior as the original monolithic file.

### 4. In-Place Navigation Index (`css/README.md`)

To prevent future agents from scanning multiple files when locating styles, `services/web-frontend/src/css/README.md` will contain a structured quick-reference:
- File descriptions and responsibilities.
- Component-to-File lookup table (e.g., "Buttons -> components.css", "Sidebar -> layout.css", "Auth Modal -> modals.css").

## Risks / Trade-offs

- **[Risk] Cascade or specificity changes causing visual regression** → *Mitigation*: Maintain rule order strictly across splits; run an automated Python verification script during tests to verify that all 115 rules, 73 classes, and 10 IDs match 1:1.
- **[Risk] Browser caching stale styles during development** → *Mitigation*: Bump asset version query parameters (`?v=1.2.0`) in `index.html` and delete the legacy `styles.css`.
