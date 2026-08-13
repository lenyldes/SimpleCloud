## 1. Environment & Infrastructure Setup

- [ ] 1.1 Update `docker-compose.yml` to include `postgres:16-alpine` service container with volume mount (`postgres_data`) and health check (`pg_isready`).
- [ ] 1.2 Add PostgreSQL database connection drivers (`pgx/v5`) to `services/storage-service/go.mod`.

## 2. Database Schema & Auto-Migrations

- [ ] 2.1 Write SQL schema migration files (`users` and `files` tables with indexes) under `internal/database/migrations/`.
- [ ] 2.2 Implement embedded SQL schema runner and PostgreSQL connection pool initializer in `internal/database/`.

## 3. Sharded File Storage Engine

- [ ] 3.1 Implement subfolder sharding path generator (`/storage/<uuid[0..1]>/<uuid[2..3]>/<uuid>`).
- [ ] 3.2 Implement disk storage engine with streaming write, on-the-fly SHA256 hash calculation, and automatic temp file cleanup on error.

## 4. API Endpoints & Streaming Handlers

- [ ] 4.1 Implement `POST /api/v1/files/upload` handler with on-the-fly quota check, returning HTTP 413 Payload Too Large on quota breach.
- [ ] 4.2 Implement `GET /api/v1/files/download/:id` streaming handler with proper headers (`Content-Disposition`, `Content-Type`, `Content-Length`).
- [ ] 4.3 Implement `GET /api/v1/files` handler returning JSON list of stored user files.

## 5. TDD Test Suite & Docker Verification

- [ ] 5.1 Write TDD unit and integration tests for storage sharding, database queries, and streaming HTTP handlers.
- [ ] 5.2 Verify Docker Compose deployment (`docker compose up`) and execute full Go test suite (`go test ./...`).
