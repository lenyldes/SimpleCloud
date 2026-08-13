## Purpose

Establishes the fundamental project guidelines, development directives for AI agents, multi-step roadmap, Go project layout, Docker Compose configuration on port 32214, and backend health status endpoint.

## ADDED Requirements

### Requirement: Agent Development Directives
The project MUST include an `AGENTS.md` file defining development standards, TDD agent roles, code modification boundaries, semantic commit formatting, and documentation rules.

#### Scenario: AI agent reads development rules
- **WHEN** an AI agent starts working on the codebase
- **THEN** it finds `AGENTS.md` defining strict TDD roles, forbidding implementation agents from editing `*_test.go` files, and requiring commit messages prefixed with `ADD:`, `UPD:`, `FIX:`, `RM:`, or `DOC:`.

### Requirement: Project Development Roadmap
The project MUST include a `ROADMAP.md` file documenting the complete 6-phase development roadmap to preserve project context across separate development sessions.

#### Scenario: Developer or AI agent checks roadmap
- **WHEN** inspecting project roadmap for future steps
- **THEN** `ROADMAP.md` contains clear details for Steps 1 through 6 (Scaffold, Postgres/UUID Storage, Auth/Users, Vanilla JS Web UI, CI/CD+Caddy, and Advanced Features).

### Requirement: Backend Health Check Endpoint
The storage service MUST provide an HTTP `GET /health` endpoint that returns a `200 OK` response with a JSON payload `{"status":"ok"}`.

#### Scenario: Health check request
- **WHEN** a GET request is sent to `/health`
- **THEN** the service responds with HTTP status 200 OK and JSON body `{"status":"ok"}`.

### Requirement: Docker Compose Configuration
The project MUST provide a `docker-compose.yml` file that runs the storage service container, mapping external host port `32214` to internal container port `8080`.

#### Scenario: Docker Compose environment launch
- **WHEN** running `docker compose up`
- **THEN** the storage service container starts and accepts HTTP connections on host port 32214.
