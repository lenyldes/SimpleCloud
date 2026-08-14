## ADDED Requirements

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
The system SHALL serve the current user profile at `GET /api/v1/auth/me` behind the authentication middleware, so an active session survives a browser page reload.

#### Scenario: Authenticated profile request
- **WHEN** an authenticated client with a valid session cookie or Bearer token sends `GET /api/v1/auth/me`
- **THEN** system SHALL return `200 OK` with the authenticated user's profile JSON (id, email, quota)

#### Scenario: Unauthenticated profile request
- **WHEN** a client without a valid session sends `GET /api/v1/auth/me`
- **THEN** system SHALL return `401 Unauthorized` and MUST NOT return any user profile data
