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
The system SHALL manage user login sessions in a `user_sessions` PostgreSQL table and issue `simplecloud_session` HTTP-only cookies for browser clients. The session cookie SHALL carry the `Secure` flag unless local development explicitly disables it via the `COOKIE_SECURE=false` environment variable, and the cookie expiration SHALL equal the configured session duration instead of a hardcoded value.

#### Scenario: Authenticated request using HTTP-only cookie
- **WHEN** client sends a request with a valid `simplecloud_session` cookie whose session is active in `user_sessions`
- **THEN** system SHALL authenticate the request, inject the associated `user_id` into the request context, and allow downstream handling

#### Scenario: Request with expired or invalid session cookie
- **WHEN** client sends a request with an expired or non-existent session cookie
- **THEN** system SHALL return `401 Unauthorized` and clear the invalid cookie

#### Scenario: Secure flag on session cookie
- **WHEN** a user logs in successfully and `COOKIE_SECURE` is unset or `true`
- **THEN** the issued `simplecloud_session` cookie SHALL have the `Secure` flag set, `HttpOnly` set, and `SameSite=Lax`.

#### Scenario: Local development without TLS
- **WHEN** a user logs in while the service runs with `COOKIE_SECURE=false`
- **THEN** the issued cookie SHALL omit the `Secure` flag so plain-HTTP local development keeps working.

#### Scenario: Cookie expiration follows session duration
- **WHEN** a user logs in successfully
- **THEN** the cookie `Expires`/`MaxAge` SHALL correspond to the configured session duration used for the `user_sessions` record, not a hardcoded constant.

### Requirement: Cross-Site Request Forgery Origin Validation
The system SHALL verify the `Origin` header on all state-changing (mutating) endpoints — login, logout, file upload, file deletion, folder creation, folder deletion. When an `Origin` header is present, its host MUST match the request host (taking `X-Forwarded-Host` into account); mismatches SHALL be rejected.

#### Scenario: Mutating request with mismatched Origin
- **WHEN** an authenticated mutating API request carries an `Origin` header whose host differs from `X-Forwarded-Host` (or `Host` when the forwarded header is absent)
- **THEN** the system SHALL respond with `403 Forbidden` and MUST NOT perform the mutation.

#### Scenario: Mutating request with same-origin Origin
- **WHEN** a mutating API request carries an `Origin` header whose host matches the request host
- **THEN** the system SHALL proceed with normal authentication and handling.

#### Scenario: Request without Origin header
- **WHEN** a mutating API request carries no `Origin` header (e.g. non-browser clients, same-page requests that omit it)
- **THEN** the CSRF check SHALL pass and the request SHALL be processed normally, with `SameSite=Lax` remaining as the second defense line.

### Requirement: Timing-Safe Login
The login flow SHALL spend comparable time on unknown-email and known-email failures so that user existence cannot be enumerated by response timing. When the email is not found, the system SHALL still perform a bcrypt comparison against a precomputed dummy hash before returning the invalid-credentials error.

#### Scenario: Login with unknown email
- **WHEN** a client submits login credentials with an email that does not exist
- **THEN** the system SHALL perform a bcrypt hash comparison (against a dummy hash) and return the same invalid-credentials error as for a wrong password, with response timing comparable to the wrong-password case.

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

### Requirement: Mandatory Database Backend at Startup
The storage service SHALL require a reachable PostgreSQL database to start and MUST terminate with a fatal startup error instead of serving requests when the database is unavailable or database configuration is missing. The codebase MUST NOT contain any mock or fallback authentication implementation with hardcoded credentials.

#### Scenario: Database unreachable at startup
- **WHEN** storage service starts and the PostgreSQL connection cannot be established (database down, wrong credentials, or timeout)
- **THEN** service SHALL exit with a fatal error and MUST NOT serve any HTTP request, including login attempts

#### Scenario: Missing database configuration
- **WHEN** storage service starts with empty or unset `POSTGRES_HOST` environment variable
- **THEN** service SHALL refuse to start with a fatal error and MUST NOT fall back to any in-memory or mock authentication mode

#### Scenario: No hardcoded credentials in production code
- **WHEN** the repository sources under `services/` are searched for fallback credentials (e.g. `adminpassword123`)
- **THEN** no production code path SHALL contain hardcoded credentials; such values MAY exist only in CI test workflow environment configuration

### Requirement: Authenticated Current User Profile Endpoint
The system SHALL serve the current user profile at `GET /api/v1/auth/me` behind the authentication middleware, so an active session survives a browser page reload. The response SHALL always include the user's current `used_bytes` and `quota_bytes` values read from the database.

#### Scenario: Authenticated profile request
- **WHEN** an authenticated client with a valid session cookie or Bearer token sends `GET /api/v1/auth/me`
- **THEN** system SHALL return `200 OK` with the authenticated user's profile JSON including `id`, `email`, `role`, `quota_bytes`, and `used_bytes` (present even when `used_bytes` is zero).

#### Scenario: Unauthenticated profile request
- **WHEN** a client without a valid session sends `GET /api/v1/auth/me`
- **THEN** system SHALL return `401 Unauthorized` and MUST NOT return any user profile data


