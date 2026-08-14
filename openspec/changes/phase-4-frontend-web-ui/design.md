## Context

See proposal.md - Why.
The backend API (`storage-service` written in Go) is operational with PostgreSQL database integration, supporting user authentication (cookie/session-based) and file CRUD endpoints (`/api/v1/auth/*`, `/api/v1/files/*`).
The web interface needs to be added as a decoupled static container service (`web-frontend`) using Vanilla HTML5, CSS custom properties adhering to `docs/DESIGN_SPEC.md`, and plain JavaScript (no heavy frontend frameworks).

## Goals / Non-Goals

**Goals:**
- Create `services/web-frontend` directory with `src/index.html`, `src/styles.css`, `src/app.js`, `Dockerfile`, and `nginx.conf`.
- Implement Vanilla CSS adhering strictly to `docs/DESIGN_SPEC.md` design tokens (Mail.ru Cloud palette with `#0077FF` primary color, Inter typography, responsive layout).
- Render Header, Left Sidebar (with quota bar), Breadcrumb toolbar, and File Workspace (Grid and List views).
- Implement Drag-and-Drop file upload overlay connected to `POST /api/v1/files/upload`.
- Implement file preview modals (Image Lightbox, Text/Code viewer, Video player) and file download/deletion context actions.
- Update `docker-compose.yml` to include `web-frontend` container proxying `/api/` requests to `storage-service:8080`.

**Non-Goals:**
- Desktop or Mobile sync client development.
- Third-party CSS framework usage (Bootstrap, Tailwind, etc.).
- Complex rich text file editing in browser (reserved for Phase 6).

## Decisions

1. **Framework Choice: Vanilla HTML/CSS/JS with Nginx**
   - *Rationale*: Keeps bundle size minimal (<100KB total), ensures lightning-fast page loading times, zero frontend build step overhead, and easy Docker deployment via `nginx:alpine`.
   - *Alternatives Considered*: React/Next.js or Vite SPA — discarded to maintain the lightweight zero-dependency ethos of SimpleCloud and comply with `AGENTS.md` and `docs/DESIGN_SPEC.md`.

2. **Decoupled Architecture with Nginx Reverse Proxy**
   - *Rationale*: `web-frontend` runs as an independent Docker container. Nginx serves static files directly and reverse proxies `/api/` requests to `storage-service:8080`. This prevents CORS issues while keeping backend and frontend independent.
   - *Alternatives Considered*: Serving static files directly from Go web server — discarded to maintain clear separation of concerns (Decoupled Microservices).

3. **CSS Design Tokens & Components System (`docs/DESIGN_SPEC.md`)**
   - *Rationale*: CSS variables in `:root` (`--color-primary: #0077FF`, `--color-bg-app: #f5f5f7`, `--font-family-base: 'Inter', ...`) defined in `src/styles.css` ensure consistent, polished aesthetics matching Mail.ru Cloud.

4. **DOM Manipulation Strategy in `app.js`**
   - *Rationale*: Modular JS functions (`renderFileList()`, `showPreviewModal()`, `initDragAndDrop()`, `updateQuotaDisplay()`) using `fetch()` with `credentials: 'include'` to handle session authentication cookies smoothly.

## Risks / Trade-offs

- **[Risk] State Management in Vanilla JS**: Complexity when updating multiple views without a reactive framework.
  - *Mitigation*: Centralize UI state in a clean lightweight state object in `app.js` (`const state = { currentPath: '/', files: [], viewMode: 'grid' }`) and trigger re-renders via explicit update functions.
- **[Risk] Cookie authentication cross-origin issues during local dev**:
  - *Mitigation*: Use Nginx in Docker Compose as single entry point (port 32214) proxying both static frontend (`/`) and API (`/api/`).
