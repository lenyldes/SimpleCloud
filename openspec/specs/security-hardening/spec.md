# Security Hardening

## Purpose

Provides HTTP response security header enforcement, rate limiting thresholds, and security hardening rules for SimpleCloud services.

## Requirements

### Requirement: Nginx Security Headers
The web frontend reverse proxy SHALL include HTTP security response headers in all responses served to clients. The `Content-Security-Policy` header SHALL NOT permit `'unsafe-inline'` for scripts; all JavaScript MUST be served from external files under the same origin.

#### Scenario: Verify presence of security headers
- **WHEN** client sends a request to any HTTP endpoint
- **THEN** response MUST include `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, and `Content-Security-Policy`.

#### Scenario: CSP blocks inline scripts
- **WHEN** a response carries the `Content-Security-Policy` header
- **THEN** the `script-src` directive MUST contain only `'self'` (no `'unsafe-inline'`), and the served pages MUST NOT rely on inline `<script>` blocks or inline event handlers.

### Requirement: HTTP Server Timeouts
The storage service HTTP server SHALL be configured with explicit read, write, and idle timeouts so that slow or stalled connections cannot exhaust server resources (Slowloris-class DoS).

#### Scenario: Slow header delivery is aborted
- **WHEN** a client opens a connection and sends request headers slower than the configured read-header timeout
- **THEN** the server SHALL close the connection instead of waiting indefinitely.

#### Scenario: Idle keep-alive connection is closed
- **WHEN** a keep-alive connection stays idle beyond the configured idle timeout
- **THEN** the server SHALL close the connection and continue serving other requests normally.

### Requirement: Request Body Size Limits
The storage service SHALL cap request body sizes on every endpoint: uploads SHALL be limited by `http.MaxBytesReader` to the user quota plus a small multipart overhead allowance, and JSON endpoints (login, folder creation) SHALL be limited to 1 MB.

#### Scenario: Oversized upload is rejected
- **WHEN** an authenticated user uploads a request body exceeding the configured body limit
- **THEN** the service SHALL respond with `HTTP 413` (or an equivalent quota/payload error status) and MUST NOT store a partial file.

#### Scenario: Oversized JSON body is rejected
- **WHEN** a client sends a JSON request body larger than 1 MB to a login or folder-creation endpoint
- **THEN** the service SHALL reject the request with a 4xx status without decoding the oversized payload.

### Requirement: Unprivileged Service Containers
Both Docker service images SHALL run their main process as a non-root user: the storage service image SHALL define and switch to a dedicated unprivileged user owning the storage directory, and the web frontend image SHALL use an unprivileged nginx base image listening on a non-privileged port.

#### Scenario: Storage service container process identity
- **WHEN** the storage-service container is running
- **THEN** the main process SHALL run as a non-root user that owns the mounted storage directory, and the service SHALL remain fully functional (upload, download, delete).

#### Scenario: Web frontend container process identity
- **WHEN** the web-frontend container is running
- **THEN** the nginx master process SHALL run as a non-root user listening on a non-privileged port, and the frontend plus API proxy SHALL remain reachable through the published host port.

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
