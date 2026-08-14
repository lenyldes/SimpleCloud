## Why

Currently, SimpleCloud provides backend REST API endpoints for user authentication and file management, but lacks a graphical user interface. Providing a modern, lightweight, fast, and responsive Vanilla HTML/CSS/JS frontend will allow users to seamlessly manage files, upload content via drag-and-drop, preview media, and manage their cloud storage directly from their web browser.

## What Changes

- Introduce a decoupled `services/web-frontend` service serving static Vanilla HTML/CSS/JS assets via Nginx.
- Design System & Styling: Vanilla CSS adhering to `docs/DESIGN_SPEC.md` and Mail.ru Cloud design tokens (`#0077FF`, modern typography, responsive layout).
- Core UI Layout: Header bar, left sidebar with navigation & quota usage indicator, breadcrumbs bar, file grid/list toggle, and sorting options.
- Dynamic File Browser: Interactive grid and list views for browsing files and folders with breadcrumb navigation.
- Drag-and-Drop Upload Overlay: Modal dropzone overlay supporting multiple file uploads directly from browser.
- Interactive Modals & Context Menus: Image lightbox viewer, text/code viewer, video player modal, and file/folder context actions (download, preview, delete, folder creation).
- REST API Integration: Seamless integration with backend API endpoints for authentication (`/api/v1/auth/*`), file upload/download (`/api/v1/files/*`), and health check (`/health`).

## Capabilities

### New Capabilities
- `web-frontend`: Web user interface providing file browsing, grid/list toggle, drag-and-drop uploading, file previews (image/text/video), breadcrumb navigation, and storage quota display.

### Modified Capabilities

## Impact

- **Frontend Codebase**: New directory `services/web-frontend/` containing `src/index.html`, `src/styles.css`, `src/app.js`, `Dockerfile`, and Nginx config.
- **Docker Compose**: Addition of `web-frontend` container service serving static files and reverse-proxying `/api/` requests to `storage-service`.
- **Backend API**: Endpoints consumed via HTTP cookie authentication.
