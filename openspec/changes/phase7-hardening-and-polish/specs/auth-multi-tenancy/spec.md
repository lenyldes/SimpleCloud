# auth-multi-tenancy delta — phase7-hardening-and-polish

## ADDED Requirements

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

## MODIFIED Requirements

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

### Requirement: Authenticated Current User Profile Endpoint
The system SHALL serve the current user profile at `GET /api/v1/auth/me` behind the authentication middleware, so an active session survives a browser page reload. The response SHALL always include the user's current `used_bytes` and `quota_bytes` values read from the database.

#### Scenario: Authenticated profile request
- **WHEN** an authenticated client with a valid session cookie or Bearer token sends `GET /api/v1/auth/me`
- **THEN** system SHALL return `200 OK` with the authenticated user's profile JSON including `id`, `email`, `role`, `quota_bytes`, and `used_bytes` (present even when `used_bytes` is zero).

#### Scenario: Unauthenticated profile request
- **WHEN** a client without a valid session sends `GET /api/v1/auth/me`
- **THEN** system SHALL return `401 Unauthorized` and MUST NOT return any user profile data
