## Context

See `proposal.md` for motivation.

SimpleCloud currently utilizes PostgreSQL 16, a Go-based `storage-service`, and Docker Compose. Admin seeding logic was introduced in Phase 3. `SeedAdminUser` in `internal/auth/service.go` previously contained fallback logic (`admin@simplecloud.local` / `adminpassword123`) when environment variables were empty.

Per explicit security architecture rules, **ZERO login credentials or passwords shall be hardcoded in Go code or configuration files**. All credentials MUST be loaded strictly from environment variables (`.env`).

## Goals / Non-Goals

**Goals:**
- Completely purge any hardcoded fallback login emails or passwords from `services/storage-service`.
- Require `ADMIN_EMAIL` and `ADMIN_PASSWORD` to be explicitly supplied via `.env` / environment variables.
- Bind `docker-compose.yml` environment variables to `.env` file variables using standard Docker Compose syntax.
- Update `.env.example` to provide placeholders for `ADMIN_EMAIL` and `ADMIN_PASSWORD` without hardcoded secrets.

**Non-Goals:**
- User self-registration endpoints (deferred to future auth phase).

## Decisions

### 1. Zero-Hardcode Enforcement in `SeedAdminUser`
- **Decision**: In `services/storage-service/internal/auth/service.go`, remove all default fallback strings (`admin@simplecloud.local`, `adminpassword123`). If `adminEmail == ""` or `adminPassword == ""`, log `[WARN] ADMIN_EMAIL or ADMIN_PASSWORD not configured. Skipping admin account seeding.` and return `nil`.
- **Rationale**: Completely prevents any unconfigured environment from spinning up default accounts.

### 2. Environment Variable Interpolation in `docker-compose.yml`
- **Decision**: Update `docker-compose.yml` services:
  ```yaml
  postgres:
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-simplecloud}
      POSTGRES_USER: ${POSTGRES_USER:-simplecloud_user}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}

  storage-service:
    environment:
      - PORT=${PORT:-8080}
      - POSTGRES_HOST=${POSTGRES_HOST:-postgres}
      - POSTGRES_PORT=${POSTGRES_PORT:-5432}
      - POSTGRES_DB=${POSTGRES_DB:-simplecloud}
      - POSTGRES_USER=${POSTGRES_USER:-simplecloud_user}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - ADMIN_EMAIL=${ADMIN_EMAIL}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
  ```

### 3. Explicit Test Credentials in Tests and CI Workflow
- **Decision**: Unit and integration test suites in `auth_db_test.go` and `ci.yml` will pass explicit test credentials as function parameters / workflow environment variables, keeping production code completely free of hardcoded accounts.

## Risks / Trade-offs

- **[Risk] Unconfigured environment has no admin user** → **Mitigation**: Clear warning in logs directing operator to populate `.env`.
