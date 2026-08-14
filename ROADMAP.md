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
  - [x] Security & secrets hardening: disabled fallback credentials across Go seeding, Docker Compose, and CI/CD.
  - [x] Nginx security hardening & rate limiting (`X-Frame-Options`, `CSP`, 5r/s login rate limit, 30r/s API rate limit).

- [x] **Phase 6: Advanced Features (Core & Navigation)**
  - [x] Backend folder hierarchy & DB schema (`folders` table, migrations, CRUD API).
  - [x] Backend storage quota enforcement & 413 checks.
  - [x] Session cleanup worker (background Go ticker for `user_sessions`).
  - [x] Frontend Login & Auth Modal UI (automatic modal on 401 Unauthorized).
  - [x] Frontend folder hierarchy navigation & path-based file browsing (`app.js` API integration, breadcrumbs).
  - [x] Frontend sidebar quota progress indicator & status threshold colors.

- [ ] **Phase 7: Manual User Testing, Bugfixing & Product Polish**
  - [x] Critical Quick Fixes & Port Hardening (Part 1 of Phase 7): fixed C5, C1, C3 in `phase7-critical-quick-fixes`.
  - [x] DB Metadata Persistence, Quotas Enforcement & Deletion Engine (Part 2 of Phase 7): fixed C2, C4, M8, H4, M7 in `phase7-db-metadata-quotas-deletion`.
  - [x] Remaining Audit Fixes from [`BUGS.md`](BUGS.md) (Part 3 of Phase 7): fixed H1-H3, H5, M1-M6, L1-L4 + fail-fast CI/CD in `phase7-hardening-and-polish`.
  - [x] Final Code Cleanup & Technical Debt (Part 4 of Phase 7): fixed database `io/fs.FS` migrations (O2), RFC 5987 `Content-Disposition` (L5), and self-hosted Inter typography / CSP in `phase7-final-cleanup`.
  - [ ] Guided User Manual Testing Protocol: `[ORCHESTRATOR-AGENT]` provides interactive step-by-step test instructions and report template for user feedback.
  - [ ] User Bug Triage & Resolution: Implement targeted fixes for all user-reported UI/UX issues and edge cases.
  - [ ] Final v1.0 Release Verification: End-to-end audit and release build ready for production use.

- [ ] **Phase 8: Future Extensions & Rich Media Features**
  - [ ] Simple text file editor (notepad in browser).
  - [ ] File and folder sharing with public links (`file_shares`) and read/read-write permissions.
  - [ ] Automated file & folder expiration worker (`expires_at`).

- [ ] **Technical Debt & Backlog**
  - [x] `internal/database` statement coverage to 85%+ (O2): inject `fs.FS` into `RunMigrations` instead of package-level `embed.FS` and test error/rollback branches (`phase7-final-cleanup`).
  - [x] Google Fonts `<link>` in `index.html` blocked by CSP `style-src 'self'`: self-host Inter typography font assets (`phase7-final-cleanup`).
  - [x] L5: `Content-Disposition` with RFC 5987 `filename*` for UTF-8 filenames (`phase7-final-cleanup`).


