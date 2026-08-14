## Why

SimpleCloud currently operates in single-tenant mode using a hardcoded default user ID (`00000000-0000-0000-0000-000000000001`). To transform SimpleCloud into a secure, production-ready cloud storage platform, we need user accounts, authentication, session management, and strict data isolation across all database queries and file handlers.

## What Changes

- **New Auth Module**: Introduce an isolated Go package `services/storage-service/internal/auth` managing user registration/login/logout, password hashing (bcrypt), session tokens, and context extraction.
- **Database Schema Updates**: 
  - Update `users` table schema to include `password_hash`, `role`, `is_active`, and updated indexes.
  - Create `user_sessions` table for server-managed session tokens (`id`, `user_id`, `token_hash`, `expires_at`, `user_agent`, `client_ip`).
- **HTTP-Only Cookie Session Auth**: Provide login/logout/me REST endpoints (`POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`) using secure `HttpOnly; SameSite=Lax` cookies (`simplecloud_session`).
- **Extensible Auth Middleware**: Provide HTTP middleware inspecting session cookies or `Authorization: Bearer` headers, validating session against DB, and injecting authenticated `user_id` into `r.Context()`.
- **Strict Multi-Tenancy Data Isolation**: Update all file storage handlers (`Upload`, `Download`, `List`) and SQL queries to strictly scope access via `WHERE user_id = $1` derived exclusively from request context.
- **Admin Startup Seeding**: Automatically seed initial admin account on service startup using environment variables (`ADMIN_EMAIL`, `ADMIN_PASSWORD`) with bcrypt password hashing.

## Capabilities

### New Capabilities
- `auth-multi-tenancy`: User authentication (session cookies + Bearer middleware), user accounts & admin seeding, user session management, and strict SQL/handler multi-tenant data isolation.

### Modified Capabilities
- `file-storage`: Enforce strict context-based `user_id` ownership on file uploads, downloads, and listing queries instead of mock user fallback.

## Impact

- **Database**: New migration `000002_auth_schema.sql` updating `users` table and adding `user_sessions` table.
- **API Endpoints**: 
  - Added: `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`.
  - Modified: All `/api/v1/files/*` endpoints now require authentication context.
- **Dependencies**: `golang.org/x/crypto/bcrypt`, `github.com/google/uuid`.
- **Environment**: New env vars `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `SESSION_DURATION_HOURS`.
