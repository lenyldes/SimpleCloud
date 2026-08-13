## 1. Documentation & Agent Rules

- [x] 1.1 Create `AGENTS.md` with strict TDD roles, permissions, commit message format rules (`ADD:`, `UPD:`, `FIX:`, `RM:`, `DOC:`), and no tautology rule.
- [x] 1.2 Create `ROADMAP.md` documenting the complete 6-phase development roadmap.

## 2. Storage Service Scaffold & TDD Tests

- [x] 2.1 Initialize Go module under `services/storage-service/` (`go.mod`).
- [x] 2.2 Write unit test `internal/handler/health_test.go` for `GET /health` endpoint following TDD.
- [ ] 2.3 Implement `internal/handler/health.go` and `cmd/main.go` to make the test pass.

## 3. Docker Compose & Environment

- [ ] 3.1 Create multi-stage `Dockerfile` in `services/storage-service/`.
- [ ] 3.2 Create `docker-compose.yml` mapping port `32214` to container `8080`.
- [ ] 3.3 Verify Docker Compose environment and test `/health` endpoint output.
