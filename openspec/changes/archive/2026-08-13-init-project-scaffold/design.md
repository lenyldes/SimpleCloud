## Context

SimpleCloud requires a minimal, clean Go backend service structure and Docker Compose setup for Step 1.
See `proposal.md` for overall motivation.

## Goals / Non-Goals

**Goals:**
- Create `AGENTS.md` containing strict TDD rules, commit rules (`ADD:`, `UPD:`, `FIX:`, `RM:`, `DOC:`), and role separation.
- Create `ROADMAP.md` documenting Steps 1 through 6 for context retention across sessions.
- Build standard Go project structure under `services/storage-service/`.
- Implement `GET /health` handler in Go.
- Create `docker-compose.yml` exposing port `32214`.
- Write unit tests for `/health` endpoint following TDD.

**Non-Goals:**
- Database integration (Postgres planned for Step 2).
- File upload/download endpoints (Step 2).
- Auth/User management (Step 3).

## Decisions

### 1. Go Project Structure (`services/storage-service`)
- **Choice**: Standard Go layout.
  - `cmd/main.go`: Entry point.
  - `internal/handler/`: HTTP handlers.
  - `Dockerfile`: Multi-stage Go build for tiny Docker image (~15MB).
- **Rationale**: Clean microservice isolation enabling future microservices to be added alongside `storage-service`.

### 2. Docker Compose & Port 32214
- **Choice**: Map host port `32214` to container port `8080`.
- **Rationale**: Non-standard port avoids collisions with other host services and integrates seamlessly with Caddy reverse proxy.

### 3. Agent Rules in `AGENTS.md`
- **Choice**:
  - Semantic commit format: `<PREFIX>: <description>` (`ADD:`, `UPD:`, `FIX:`, `RM:`, `DOC:`).
  - TDD rules: Test Agent writes tests, Code Agent makes them pass without modifying test files.
  - No tautologies in commit messages.

## Risks / Trade-offs

- [Risk] Docker build failure on host → Mitigation: Use standard `golang:1.22-alpine` builder image.
