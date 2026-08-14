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

### Requirement: Caddy Reverse Proxy Docker Network Integration
The system SHALL connect the `web-frontend` container to the external `caddy-public` Docker network to allow Caddy reverse proxy routing without host port binding.

#### Scenario: Routing traffic to web-frontend via Caddy
- **WHEN** an HTTPS request is received by Caddy for `test-cloud.lenyldes.ru`
- **THEN** Caddy proxies the request directly to `simplecloud-web-frontend:80` inside the `caddy-public` Docker network.
