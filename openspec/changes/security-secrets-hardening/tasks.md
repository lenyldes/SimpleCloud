## 1. Unit & Integration Tests for Hardened Seeding

- [x] 1.1 `[TEST-AGENT]` Write unit and integration tests in `services/storage-service/internal/auth/auth_db_test.go` verifying that `SeedAdminUser` skips seeding when `adminEmail` or `adminPassword` are empty, and successfully seeds when non-empty parameters are explicitly supplied.

## 2. Go Backend Hardening Implementation

- [ ] 2.1 `[CODE-AGENT]` Remove fallback default hardcoded credentials (`adminpassword123`) from `services/storage-service/internal/auth/service.go` in `SeedAdminUser`, logging a warning and returning early when env vars are missing.

## 3. Configuration & CI/CD Hardening

- [ ] 3.1 `[CODE-AGENT]` Update `docker-compose.yml` and `.env.example` to require `ADMIN_EMAIL`, `ADMIN_PASSWORD`, and `POSTGRES_PASSWORD` from `.env`.
- [ ] 3.2 `[CODE-AGENT]` Update `.github/workflows/ci.yml` test step to pass explicit non-production test credentials in environment variables.
