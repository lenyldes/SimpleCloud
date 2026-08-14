# SimpleCloud - Development Roadmap

This document outlines the step-by-step master development plan for SimpleCloud. Each phase is implemented incrementally via OpenSpec changes.

---

- [x] **Phase 1: Project Scaffold & Foundation**
  - [x] `AGENTS.md` rules (TDD roles, commit message conventions, coding standards).
  - [x] Master `ROADMAP.md` for context persistence across sessions.
  - [x] Go project structure (`services/storage-service/`).
  - [x] Basic `GET /health` endpoint with TDD unit tests.
  - [x] `docker-compose.yml` mapped to custom host port `32214`.

- [x] **Phase 2: Database & File Storage Core**
  - [x] PostgreSQL 16 container in Docker Compose (`image: postgres:16-alpine`).
  - [x] Database schema: `users` (quotas), `files` (UUID, path, size, mime_type, sha256, expires_at).
  - [x] Binary file storage on disk with subfolder sharding (`/storage/f4/7a/<uuid>`).
  - [x] API endpoints: `POST /api/v1/files/upload`, `GET /api/v1/files/download/:id`, `GET /api/v1/files`.
  - [x] TDD unit & integration test suite.

- [x] **Phase 3: Authentication & Multi-Tenancy**
  - [x] User accounts schema (`users`, `user_sessions`), password hashing (bcrypt), and seeding initial admin user.
  - [x] Session authentication using secure HTTP-only cookies (Web UI) with extensible middleware supporting `Authorization: Bearer`.
  - [x] Strict user isolation (`WHERE user_id = $1` on all SQL queries and handlers).

- [x] **Phase 4: Frontend Web UI (Vanilla JS)**
  - [x] Ultra-lightweight Vanilla HTML/CSS/JS frontend served via Go / Nginx container.
  - [x] File and folder grid/list views with breadcrumb navigation.
  - [x] Drag-and-Drop file & folder upload interface.
  - [x] File preview (text, images, video) and download capabilities.

- [x] **Phase 5: CI/CD & Deployment**
  - [x] GitHub Actions workflow (`ci.yml`): Go linting (`golangci-lint`) and TDD automated test execution on PRs.
  - [x] Auto-deployment workflow (`deploy.yml`): SSH auto-deployment to remote VPS upon merging into `main`.
  - [x] Caddy reverse proxy integration (`test-cloud.lenyldes.ru` -> `localhost:32214`) with automatic Let's Encrypt HTTPS certificates.

- [x] **Phase 6: Advanced Features (Core & Navigation)**
  - [x] Frontend Login & Auth Modal UI (automatic modal on 401 Unauthorized).
  - [x] Nested folder hierarchy navigation & path-based file browsing (`folders` DB table, CRUD API, breadcrumbs).
  - [x] Storage quota enforcement per user (Backend 413 check & Frontend quota usage indicator).
  - [x] Automated session expiration worker (Background Go ticker for cleaning expired `user_sessions`).
  - [x] Changelog API endpoint (`/api/v1/sync/changes`) for future desktop/mobile sync clients.
  - [x] Custom SVG icons and refined UI design.

- [ ] **Phase 7: Manual User Testing, Bugfixing & Product Polish**
  - [ ] Guided User Manual Testing Protocol: `[ORCHESTRATOR-AGENT]` provides interactive step-by-step test instructions and report template for user feedback.
  - [ ] User Bug Triage & Resolution: Implement targeted fixes for all user-reported UI/UX issues and edge cases.
  - [ ] Final v1.0 Release Verification: End-to-end audit and release build ready for production use.

- [ ] **Phase 8: Future Extensions & Rich Media Features**
  - [ ] Simple text file editor (notepad in browser).
  - [ ] File and folder sharing with public links (`file_shares`) and read/read-write permissions.
  - [ ] Automated file & folder expiration worker (`expires_at`).

