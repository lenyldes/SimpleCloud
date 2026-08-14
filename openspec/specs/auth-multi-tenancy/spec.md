# Auth Multi-Tenancy Specification

## Purpose

Provides secure user authentication, session management via HTTP-only cookies and Bearer tokens, initial admin seeding, and strict multi-tenant data isolation across all cloud storage operations.

## Requirements

### Requirement: User Account Schema and Admin Seeding
The system MUST support user accounts stored in PostgreSQL with bcrypt-hashed passwords and MUST seed an initial administrator account on startup ONLY IF both `ADMIN_EMAIL` and `ADMIN_PASSWORD` environment variables are explicitly provided via environment configuration (`.env`). The codebase MUST NOT contain any hardcoded fallback login emails or passwords.

#### Scenario: Successful admin seeding on startup
- **WHEN** storage service starts with non-empty `ADMIN_EMAIL` and `ADMIN_PASSWORD` environment variables set
- **THEN** system SHALL check if an admin user exists, and if missing, insert the admin user with a bcrypt-hashed password derived strictly from `ADMIN_PASSWORD` and default 50 GB storage quota

#### Scenario: Skipping admin seeding when env vars are missing
- **WHEN** storage service starts with empty or missing `ADMIN_EMAIL` or `ADMIN_PASSWORD` environment variables
- **THEN** system SHALL log a warning stating that admin seeding was skipped due to missing environment variables and MUST NOT insert any default or fallback admin credentials into PostgreSQL

#### Scenario: Admin login with valid credentials
- **WHEN** user sends `POST /api/v1/auth/login` with correct email and password
- **THEN** system SHALL verify the bcrypt hash, issue a new session token, set a secure `HttpOnly; SameSite=Lax` cookie `simplecloud_session`, and return `200 OK` with user profile JSON

#### Scenario: Failed login with invalid credentials
- **WHEN** user sends `POST /api/v1/auth/login` with incorrect password or non-existent email
- **THEN** system SHALL return `401 Unauthorized` with an error message and MUST NOT set a session cookie

### Requirement: HTTP-Only Cookie Session Authentication
The system SHALL manage user login sessions in a `user_sessions` PostgreSQL table and issue `simplecloud_session` HTTP-only cookies for browser clients.

#### Scenario: Authenticated request using HTTP-only cookie
- **WHEN** client sends a request with a valid `simplecloud_session` cookie whose session is active in `user_sessions`
- **THEN** system SHALL authenticate the request, inject the associated `user_id` into the request context, and allow downstream handling

#### Scenario: Request with expired or invalid session cookie
- **WHEN** client sends a request with an expired or non-existent session cookie
- **THEN** system SHALL return `401 Unauthorized` and clear the invalid cookie

### Requirement: Extensible Auth Middleware
The system SHALL provide an HTTP middleware that extracts authentication credentials from either `simplecloud_session` cookie OR `Authorization: Bearer <token>` header, resolving the user identity into `r.Context()`.

#### Scenario: Authenticated request using Bearer token
- **WHEN** client sends a request with header `Authorization: Bearer <valid_token_hash>`
- **THEN** middleware SHALL resolve the active session from database, inject `user_id` into context, and proceed to handler

#### Scenario: Unauthenticated request to protected endpoint
- **WHEN** client sends a request to `/api/v1/files/*` without valid cookie or Bearer header
- **THEN** middleware SHALL immediately abort execution and return `401 Unauthorized`

### Requirement: Multi-Tenancy Data Isolation
The system MUST enforce strict user data isolation across all database operations (`Upload`, `Download`, `List`, `Delete`) using the `user_id` injected into `r.Context()`.

#### Scenario: Listing files returns only current user's files
- **WHEN** authenticated User A requests `GET /api/v1/files`
- **THEN** system SHALL execute SQL query containing `WHERE user_id = $1` with User A's ID and return only User A's files

#### Scenario: User B attempts to download User A's file
- **WHEN** authenticated User B requests `GET /api/v1/files/download/<file_id_belonging_to_user_A>`
- **THEN** system SHALL execute SQL query scoped with `WHERE id = $1 AND user_id = $2`, find no match for User B, and return `440 Not Found` or `404 Not Found` without disclosing file existence

### Requirement: User Logout and Session Revocation
The system SHALL support session invalidation on logout, removing or deactivating the session in `user_sessions` and expiring the HTTP cookie.

#### Scenario: User logs out successfully
- **WHEN** authenticated user sends `POST /api/v1/auth/logout`
- **THEN** system SHALL delete or deactivate the session in `user_sessions`, set the `simplecloud_session` cookie expiration to past time, and return `200 OK`

### Requirement: Session Expiration Background Garbage Collector
The system SHALL run a background Go ticker worker in `storage-service` (running at a configurable interval, default 1 minute) to automatically purge expired user sessions (`expires_at < NOW()`) from the `user_sessions` PostgreSQL table.

#### Scenario: Background session purging
- **WHEN** ticker interval elapses and there are sessions in `user_sessions` with `expires_at < CURRENT_TIMESTAMP`
- **THEN** background worker SHALL execute `DELETE FROM user_sessions WHERE expires_at < CURRENT_TIMESTAMP` and log the count of purged sessions.

