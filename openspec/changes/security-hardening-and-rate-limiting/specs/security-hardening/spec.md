## Purpose

Provides HTTP response security header enforcement, rate limiting thresholds, and security hardening rules for SimpleCloud services.

## ADDED Requirements

### Requirement: Nginx Security Headers
The web frontend reverse proxy SHALL include HTTP security response headers in all responses served to clients.

#### Scenario: Verify presence of security headers
- **WHEN** client sends a request to any HTTP endpoint
- **THEN** response MUST include `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, and `Content-Security-Policy`.

### Requirement: Authentication Endpoint Rate Limiting
The reverse proxy SHALL enforce rate limiting on authentication login requests to prevent password brute-force attacks.

#### Scenario: Exceeding login rate limit
- **WHEN** a client sends more than 5 requests per second to `/api/v1/auth/login`
- **THEN** the server MUST respond with `HTTP 429 Too Many Requests`.

### Requirement: General API Rate Limiting
The reverse proxy SHALL enforce general rate limiting on API requests to protect server resources against DDoS.

#### Scenario: Exceeding general API rate limit
- **WHEN** a client exceeds 30 requests per second across `/api/` endpoints
- **THEN** the server MUST respond with `HTTP 429 Too Many Requests`.

### Requirement: Secure Go Base Image
The storage service Docker container build SHALL use an updated base image without known standard library vulnerability advisories.

#### Scenario: Dependency vulnerability check
- **WHEN** `govulncheck` is executed against `services/storage-service`
- **THEN** zero standard library vulnerabilities SHALL be reported for active call paths.
