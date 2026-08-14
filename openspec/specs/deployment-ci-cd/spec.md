## Purpose

Automates continuous integration code quality checks, automated unit/integration test suite execution, and secure deployment of SimpleCloud to VPS.

## Requirements

### Requirement: Automated Continuous Integration
The system SHALL run linting and Go unit/integration test suites on every pull request and push to `main` in GitHub Actions, explicitly injecting non-production test environment variables.

#### Scenario: Successful CI pipeline run
- **WHEN** a pull request or commit push to `main` occurs
- **THEN** GitHub Actions runs `golangci-lint` and `go test -v -cover ./...` against a live PostgreSQL service container with explicit test environment variables (`ADMIN_EMAIL`, `ADMIN_PASSWORD`, `POSTGRES_PASSWORD`) set in the test runner workflow and verifies 85%+ statement coverage.

### Requirement: Strict Environment Variable Configuration in Docker Compose
The system SHALL require database and initial admin secrets to be loaded from environment variables in `docker-compose.yml`, using `.env` values or explicitly declared environment parameters without unencrypted production fallbacks in codebase.

#### Scenario: Docker Compose container startup
- **WHEN** `docker compose up -d` is executed
- **THEN** system SHALL interpolate `${POSTGRES_PASSWORD}`, `${ADMIN_EMAIL}`, and `${ADMIN_PASSWORD}` from `.env` or system environment into `postgres` and `storage-service` containers.

### Requirement: Automated Continuous Deployment
The system SHALL execute automated SSH deployment to `pi5server` VPS upon merging code into `main` after CI checks pass.

#### Scenario: Successful CD deployment to VPS
- **WHEN** code is merged into `main`
- **THEN** GitHub Actions SSHs into `pi5server`, pulls the latest code, and runs `docker compose up -d --build` to update services.

### Requirement: Fail-Fast Pipeline Ordering
The CI/CD workflow SHALL NOT execute any downstream job when an upstream job failed or was cancelled: deployment SHALL require successful lint and test jobs, and jobs that do not declare explicit dependencies SHALL be gated so a failed earlier stage prevents later stages from running.

#### Scenario: Lint failure blocks tests and deploy
- **WHEN** the lint job fails on a push to `main`
- **THEN** the test and deploy jobs SHALL NOT run (skipped or never started), and no deployment SHALL occur.

#### Scenario: Test failure blocks deploy
- **WHEN** the lint job passes but the test job fails
- **THEN** the deploy job SHALL NOT run and the production server SHALL not be touched.

### Requirement: Deterministic Dependency Build Layer
The storage service Docker build SHALL copy both `go.mod` and `go.sum` into the build stage before running `go mod download`, so module downloads are verified against recorded checksums and the dependency layer is cache-deterministic.

#### Scenario: Docker build downloads verified dependencies
- **WHEN** the storage service image is built
- **THEN** the build SHALL execute `go mod download` only after both `go.mod` and `go.sum` are present in the build context working directory, and the build SHALL succeed with verified module checksums.

### Requirement: Caddy Reverse Proxy Docker Network Integration
The system SHALL connect the `web-frontend` container to the external `caddy-public` Docker network to allow Caddy reverse proxy routing without host port binding.

#### Scenario: Routing traffic to web-frontend via Caddy
- **WHEN** an HTTPS request is received by Caddy for `test-cloud.lenyldes.ru`
- **THEN** Caddy proxies the request directly to `simplecloud-web-frontend:80` inside the `caddy-public` Docker network.

### Requirement: No Host-Published Backend Ports
The Docker Compose deployment MUST NOT publish the `storage-service` API port or the PostgreSQL port on the host. All external client traffic SHALL enter exclusively through the `web-frontend` reverse proxy (host port `32214` / Caddy), so nginx rate limiting and request size limits cannot be bypassed and the database is not reachable outside the Docker network.

#### Scenario: Direct backend port access refused
- **WHEN** a client outside the Docker host network attempts to connect to the former backend ports (`8080` storage API, `5432` PostgreSQL) on the server
- **THEN** the connection SHALL be refused because no host port mapping exists

#### Scenario: Application remains reachable through the frontend proxy
- **WHEN** `docker compose up -d` is executed after the port mappings are removed
- **THEN** the application SHALL continue to serve login, upload, download, and browsing flows through `web-frontend` on host port `32214` and via the Caddy-routed domain, with `storage-service` reachable from other containers only by its Docker network service name

