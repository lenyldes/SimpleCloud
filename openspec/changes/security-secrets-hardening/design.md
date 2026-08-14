## Context

See `proposal.md` for motivation.

SimpleCloud currently utilizes PostgreSQL 16, a Go-based `storage-service`, and Docker Compose. Admin seeding logic was introduced in Phase 3. However, `SeedAdminUser` in `internal/auth/service.go` contained a fallback mechanism that defaulted `adminEmail` to `admin@simplecloud.local` and `adminPassword` to `adminpassword123` when environment variables were empty.

## Goals / Non-Goals

**Goals:**
- Completely eliminate hardcoded fallback values for admin credentials in Go code.
- Bind `docker-compose.yml` environment variables to `.env` file values using standard Docker Compose variable interpolation.
- Update `.env.example` to provide explicit template entries for `ADMIN_EMAIL` and `ADMIN_PASSWORD`.
- Ensure integration and unit tests supply explicit non-production test credentials without relying on code fallbacks.

**Non-Goals:**
- Implementing external secret vaults (HashiCorp Vault, AWS Secrets Manager). Standard `.env` and environment variables satisfy all self-hosting requirements.

## Decisions

### 1. Remove Hardcoded Credentials in `SeedAdminUser`
- **Decision**: Modify `services/storage-service/internal/auth/service.go` so that `SeedAdminUser` checks if `adminEmail` or `adminPassword` are empty. If either is empty, it returns `nil` early and logs `[WARN] ADMIN_EMAIL or ADMIN_PASSWORD not set. Skipping admin account seeding.`
- **Rationale**: Prevents any accidental seeding of default insecure accounts in misconfigured environments.

### 2. Environment Variable Interpolation in `docker-compose.yml`
- **Decision**: Update `docker-compose.yml` services:
  ```yaml
  postgres:
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-simplecloud}
      POSTGRES_USER: ${POSTGRES_USER:-simplecloud_user}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-simplecloud_dev_password}

  storage-service:
    environment:
      - PORT=${PORT:-8080}
      - POSTGRES_HOST=${POSTGRES_HOST:-postgres}
      - POSTGRES_PORT=${POSTGRES_PORT:-5432}
      - POSTGRES_DB=${POSTGRES_DB:-simplecloud}
      - POSTGRES_USER=${POSTGRES_USER:-simplecloud_user}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-simplecloud_dev_password}
      - ADMIN_EMAIL=${ADMIN_EMAIL}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
  ```
- **Rationale**: Uses Docker Compose standard variable resolution while allowing default fallbacks for local dev `postgres` container passwords while requiring explicit `ADMIN_EMAIL` and `ADMIN_PASSWORD` in `.env`.

### 3. Explicit Test Credentials in `auth_db_test.go` and `ci.yml`
- **Decision**: Update unit/integration tests in `internal/auth/auth_db_test.go` to explicitly pass test credentials (`admin@simplecloud.local`, `adminpassword123`) to `SeedAdminUser`, and configure `ci.yml` test steps to pass `ADMIN_EMAIL` and `ADMIN_PASSWORD` env vars.

## Risks / Trade-offs

- **[Risk] Existing deployments without `.env` won't seed admin user automatically** → **Mitigation**: Update `.env.example` and release notes/documentation so operators know to populate `.env`.
