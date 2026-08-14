## 1. Database Schema & Migration

- [x] 1.1 `[TEST-AGENT]` Write integration test for database migration `000002_auth_schema.sql` validating `users` schema updates and `user_sessions` table creation.
- [x] 1.2 `[CODE-AGENT]` Add migration file `services/storage-service/internal/database/migrations/000002_auth_schema.sql`.

## 2. Auth Package Core (`internal/auth`)

- [x] 2.1 `[TEST-AGENT]` Write unit tests in `services/storage-service/internal/auth/auth_test.go` for bcrypt password hashing, session token creation, and typed context helpers (`WithUserID`, `GetUserIDFromContext`).
- [ ] 2.2 `[CODE-AGENT]` Implement `services/storage-service/internal/auth/context.go` and `services/storage-service/internal/auth/service.go`.

## 3. Auth HTTP Handlers & Middleware (`internal/auth`)

- [x] 3.1 `[TEST-AGENT]` Write HTTP integration tests for `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`, and `RequireAuth` middleware (testing Cookie and Bearer header resolution).
- [ ] 3.2 `[CODE-AGENT]` Implement `services/storage-service/internal/auth/handler.go` and `services/storage-service/internal/auth/middleware.go`.

## 4. Admin Seeding & Multi-Tenancy File Handlers

- [x] 4.1 `[TEST-AGENT]` Write integration tests for startup admin account seeding (`ADMIN_EMAIL`/`ADMIN_PASSWORD`) and strict multi-tenant file scoping (`WHERE user_id = $1` on uploads, downloads, listings).
- [ ] 4.2 `[CODE-AGENT]` Implement admin startup seeding in `main.go` / `InitDB` and update `services/storage-service/internal/handler/file.go` to use context `user_id`.

## 5. Verification & Compliance Audit

- [ ] 5.1 `[AUDIT-AGENT]` Audit code quality, test coverage threshold (85%+ statement coverage across `internal/*`), formatting (`gofmt`), security hygiene, and Docker container build.
