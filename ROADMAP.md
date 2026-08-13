# SimpleCloud - Development Roadmap

This document outlines the step-by-step master development plan for SimpleCloud. Each phase is implemented incrementally via OpenSpec changes.

---

## Phase 1: Project Scaffold & Foundation (Current Phase)
- `AGENTS.md` rules (TDD roles, commit message conventions, coding standards).
- Master `ROADMAP.md` for context persistence across sessions.
- Go project structure (`services/storage-service/`).
- Basic `GET /health` endpoint.
- `docker-compose.yml` mapped to custom host port `32214`.

## Phase 2: Database & File Storage Core
- PostgreSQL 16 container in Docker Compose (`image: postgres:16-alpine`).
- Database schema: `users` (quotas), `files` (UUID, path, size, mime_type, sha256, expires_at).
- Binary file storage on disk with subfolder sharding (`/storage/f4/7a/<uuid>`).
- API endpoints: `POST /api/v1/files/upload`, `GET /api/v1/files/download/:id`, `GET /api/v1/files`.
- TDD unit & integration test suite.

## Phase 3: Authentication & Multi-Tenancy
- User accounts table and seeding initial admin user.
- Authentication using secure HTTP-only cookies (Web UI) and Bearer JWT / API Tokens (external clients).
- Strict user isolation (`WHERE user_id = $1` on all SQL queries).

## Phase 4: Frontend Web UI (Vanilla JS)
- Ultra-lightweight Vanilla HTML/CSS/JS frontend served via Go / Nginx container.
- File and folder grid/list views with breadcrumb navigation.
- Drag-and-Drop file & folder upload interface.
- File preview (text, images, video) and download capabilities.

## Phase 5: CI/CD & Deployment
- GitHub Actions workflow (`ci.yml`): Go linting (`golangci-lint`) and TDD automated test execution on PRs.
- Auto-deployment workflow (`deploy.yml`): SSH auto-deployment to remote VPS upon merging into `main`.
- Caddy reverse proxy integration (`cloud.lenyldes.ru` -> `localhost:32214`) with automatic Let's Encrypt HTTPS certificates.

## Phase 6: Advanced Features
- Simple text file editor (notepad in browser).
- File and folder sharing with public links and read/read-write permissions.
- Storage quota enforcement per user.
- Automated file & folder expiration worker (`expires_at`).
- Changelog API endpoint (`/api/v1/sync/changes`) for future desktop/mobile sync clients.
- Custom SVG icons and refined UI design.
