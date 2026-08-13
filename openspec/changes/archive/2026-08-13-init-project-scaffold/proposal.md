## Why

Starting development of SimpleCloud - a lightweight, super-fast, clean self-hosted cloud storage web application.
We need an initial, robust foundation (scaffold) that establishes strict AI development rules (TDD, role separation between test and implementation agents, semantic commits), records the overall project roadmap, and sets up a Docker Compose environment listening on port 32214 with a basic Go backend.

## What Changes

- Create `AGENTS.md` containing strict guidelines for AI agents: TDD workflow rules (Test agent vs Code agent permissions), commit message formats (`ADD:`, `UPD:`, `FIX:`, `RM:`, `DOC:`), Go formatting, and documentation rules.
- Create `ROADMAP.md` documenting the full 6-step project development plan so future sessions retain context.
- Set up standard Go project layout for `storage-service` under `services/storage-service/`.
- Implement a basic Go HTTP server with a `/health` endpoint returning `200 OK`.
- Create `docker-compose.yml` exposing the service on custom host port `32214`.
- Add initial TDD unit/integration tests for `/health` endpoint.

## Capabilities

### New Capabilities
- `project-scaffold`: Initial project setup, AGENTS.md development rules, ROADMAP.md development plan, Go storage-service scaffold, and Docker Compose configuration.

### Modified Capabilities

None.

## Impact

- Creates initial project root structure, `AGENTS.md`, `ROADMAP.md`, `docker-compose.yml`, and `services/storage-service/`.
- Exposes host port `32214` mapped to container `8080`.
