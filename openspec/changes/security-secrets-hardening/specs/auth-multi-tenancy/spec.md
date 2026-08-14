## MODIFIED Requirements

### Requirement: User Account Schema and Admin Seeding
The system MUST support user accounts stored in PostgreSQL with bcrypt-hashed passwords and MUST seed an initial administrator account on startup ONLY IF both `ADMIN_EMAIL` and `ADMIN_PASSWORD` environment variables are explicitly provided. The system MUST NOT fall back to any hardcoded default passwords or emails.

#### Scenario: Successful admin seeding on startup
- **WHEN** storage service starts with non-empty `ADMIN_EMAIL` and `ADMIN_PASSWORD` environment variables set
- **THEN** system SHALL check if an admin user exists, and if missing, insert the admin user with a bcrypt-hashed password derived from `ADMIN_PASSWORD` and default 50 GB storage quota

#### Scenario: Skipping admin seeding when env vars are missing
- **WHEN** storage service starts with empty or missing `ADMIN_EMAIL` or `ADMIN_PASSWORD` environment variables
- **THEN** system SHALL log a warning stating that admin seeding was skipped due to missing environment variables and MUST NOT insert any default admin credentials into PostgreSQL

#### Scenario: Admin login with valid credentials
- **WHEN** user sends `POST /api/v1/auth/login` with correct email and password
- **THEN** system SHALL verify the bcrypt hash, issue a new session token, set a secure `HttpOnly; SameSite=Lax` cookie `simplecloud_session`, and return `200 OK` with user profile JSON

#### Scenario: Failed login with invalid credentials
- **WHEN** user sends `POST /api/v1/auth/login` with incorrect password or non-existent email
- **THEN** system SHALL return `401 Unauthorized` with an error message and MUST NOT set a session cookie
