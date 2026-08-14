## MODIFIED Requirements

### Requirement: Automated Continuous Integration
The system SHALL run linting and Go unit/integration test suites on every pull request and push to `main` in GitHub Actions, explicitly injecting non-production test environment variables.

#### Scenario: Successful CI pipeline run
- **WHEN** a pull request or commit push to `main` occurs
- **THEN** GitHub Actions runs `golangci-lint` and `go test -v -cover ./...` against a live PostgreSQL service container with explicit test environment variables (`ADMIN_EMAIL`, `ADMIN_PASSWORD`, `POSTGRES_PASSWORD`) set in the test runner workflow and verifies 85%+ statement coverage.

## ADDED Requirements

### Requirement: Strict Environment Variable Configuration in Docker Compose
The system SHALL require database and initial admin secrets to be loaded from environment variables in `docker-compose.yml`, using `.env` values or explicitly declared environment parameters without unencrypted production fallbacks in codebase.

#### Scenario: Docker Compose container startup
- **WHEN** `docker compose up -d` is executed
- **THEN** system SHALL interpolate `${POSTGRES_PASSWORD}`, `${ADMIN_EMAIL}`, and `${ADMIN_PASSWORD}` from `.env` or system environment into `postgres` and `storage-service` containers.
