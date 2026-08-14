## 1. Test Agent Tasks (RED Phase)

- [x] 1.1 Add frontend integration test suite verifying static asset delivery and proxy endpoint accessibility

## 2. Web Frontend Structure & Docker Setup (GREEN Phase)

- [ ] 2.1 Create `services/web-frontend` directory structure with `src/index.html`, `src/styles.css`, `src/app.js`, `Dockerfile`, and `nginx.conf`
- [ ] 2.2 Configure Nginx static asset serving and API reverse proxy routing to `storage-service:8080`

## 3. UI Design System & Component Layout (GREEN Phase)

- [ ] 3.1 Implement design tokens, variables, typography, and responsive rules in `styles.css` adhering to `docs/DESIGN_SPEC.md`
- [ ] 3.2 Construct complete HTML component skeleton in `index.html` (Header bar, Left Sidebar with Quota bar, Breadcrumbs toolbar, Dropzone overlay, Workspace container)

## 4. Interactive Logic, API Integration & Modals (GREEN Phase)

- [ ] 4.1 Implement dynamic REST API interaction and workspace rendering in `app.js` (Grid/List views, Breadcrumb navigation, File sorting)
- [ ] 4.2 Implement Drag-and-Drop file upload interface and progress notifications connected to `/api/v1/files/upload`
- [ ] 4.3 Implement interactive modals (Image Lightbox, Text/Code viewer, Video player) and context operations (Download, Delete, New Folder creation)
- [ ] 4.4 Update `docker-compose.yml` to launch `web-frontend` container service mapped to host port 32214
