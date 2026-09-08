# project-scaffold Specification

## Purpose
Establishes the fundamental project guidelines, development directives for AI agents, multi-step roadmap, Go project layout, Docker Compose configuration on port 32214, and backend health status endpoint.
## Requirements
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

### Requirement: Graceful Service Shutdown
The storage service SHALL handle termination signals (`SIGTERM`, `SIGINT`) by gracefully shutting down the HTTP server within a bounded timeout (allowing in-flight requests, including uploads, to finish), then closing the database connection pool before exiting.

#### Scenario: SIGTERM during active upload
- **WHEN** the service receives `SIGTERM` while an upload request is in progress
- **THEN** the service SHALL stop accepting new connections, allow the in-flight request to complete (within the shutdown timeout), close the database pool, and exit cleanly with a zero/success exit status.

#### Scenario: Shutdown timeout expires
- **WHEN** in-flight requests do not finish within the configured shutdown timeout
- **THEN** the service SHALL force shutdown and exit rather than hanging indefinitely.

#### Scenario: Startup failure surfaces immediately
- **WHEN** the HTTP server fails to start (e.g. port already in use)
- **THEN** the service SHALL log the error and terminate with a non-zero exit status.

### Requirement: Docker Compose Configuration
The project MUST provide a `docker-compose.yml` file that runs the storage service container, mapping external host port `32214` to internal container port `8080`.

#### Scenario: Docker Compose environment launch
- **WHEN** running `docker compose up`
- **THEN** the storage service container starts and accepts HTTP connections on host port 32214.

### Requirement: Universal File Line Limit Architecture Guard
The project SHALL enforce an automated architecture guard ensuring that every readable text-based or human-readable file across the repository does not exceed 450 lines of code or text. This constraint SHALL apply by universal principle to all readable file formats (including `.go`, `.js`, `.css`, `.html`, `.md`, `.sql`, `.sh`, `.yml`, `.yaml`, `.conf`, Dockerfile, and arbitrary future text extensions) and SHALL exclude only strictly non-text binary files (detected via known binary extensions such as `.woff2` and images, or presence of null bytes `0x00` and invalid UTF-8) and `.git/` metadata.

#### Scenario: Architecture test enforces file length ceiling
- **WHEN** the automated test suite executes `TestMaxFileLineCount`
- **THEN** every tracked text file in the repository SHALL contain 450 or fewer lines of text, and any file exceeding 450 lines SHALL cause the test suite to fail with the file path and exact line count.

#### Scenario: Binary files and git directory are excluded from line count guard
- **WHEN** the architecture guard inspects tracked repository assets
- **THEN** binary files containing null bytes `0x00` or known binary extensions (e.g. `.woff2`) and `.git/` metadata SHALL be skipped without triggering false failures.

