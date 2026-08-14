## Why

A critical security vulnerability was identified where the Go `storage-service` automatically seeds an initial admin account with a hardcoded fallback password (`adminpassword123`) whenever `ADMIN_PASSWORD` environment variable is not explicitly provided. Furthermore, `docker-compose.yml` lacked `ADMIN_EMAIL` and `ADMIN_PASSWORD` container environment bindings and contained hardcoded database credentials (`simplecloud_dev_password`), resulting in production deployments automatically spinning up vulnerable default admin credentials. This emergency change hardens security by removing hardcoded credentials fallback from backend logic, requiring explicit environment configuration via `.env`, and updating docker-compose and CI/CD pipelines.

## What Changes

- **Backend Logic (Go)**: Remove insecure fallback default admin credentials (`admin@simplecloud.local` / `adminpassword123`) in `SeedAdminUser`. Skip admin account seeding completely if `ADMIN_EMAIL` or `ADMIN_PASSWORD` environment variables are empty or missing, logging an explicit warning.
- **Docker Compose**: Update `docker-compose.yml` to bind database credentials (`POSTGRES_PASSWORD`) and admin seeding environment variables (`ADMIN_EMAIL`, `ADMIN_PASSWORD`) to `.env` variables with explicit variable interpolation.
- **Environment Template**: Update `.env.example` to include mandatory `ADMIN_EMAIL` and `ADMIN_PASSWORD` template fields, along with clear security guidance.
- **CI/CD Pipeline**: Update GitHub Actions workflow (`ci.yml`) to inject explicit non-production test credentials into test environments so integration tests pass cleanly without relying on hardcoded code fallbacks.

## Capabilities

### Modified Capabilities

- `auth-multi-tenancy`: Enforce mandatory environment variables (`ADMIN_EMAIL`, `ADMIN_PASSWORD`) for admin seeding and remove fallback default passwords.
- `deployment-ci-cd`: Enforce mandatory environment variable loading and `.env` template requirements across Docker Compose and GitHub Actions CI/CD workflows.

## Impact

- **Affected Code**: `services/storage-service/cmd/main.go`, `services/storage-service/internal/auth/service.go`, `services/storage-service/internal/auth/auth_db_test.go`, `docker-compose.yml`, `.env.example`, `.github/workflows/ci.yml`.
- **Security**: Completely eliminates zero-day vulnerability where an unconfigured deployment spins up an admin user with a known hardcoded password.
- **Compatibility**: Requires production operators to supply `.env` or set environment variables before running `docker compose up`.
