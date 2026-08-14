## Why

Following an automated local security audit of SimpleCloud, several security and resilience gaps were identified in the reverse proxy and Docker environment:
1. Missing Rate Limiting on critical authentication endpoints (`/api/v1/auth/login`) and general API routes, exposing the application to credential brute-forcing and DDoS attacks.
2. Missing HTTP Security Headers (`X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Content-Security-Policy`) in Nginx responses, exposing users to clickjacking and MIME-sniffing vulnerabilities.
3. Outdated standard library packages in the Go storage-service base image flagged by `govulncheck`.

Hardening these configurations in Nginx and Docker will ensure high baseline security, resilience against brute-force attacks, and compliance with production security standards before public v1.0 release.

## What Changes

- **Rate Limiting in Nginx (`web-frontend/nginx.conf`)**:
  - Define `login_limit` zone (5 requests/second, burst=5, nodelay) for `/api/v1/auth/login` returning HTTP 429 when breached.
  - Define `api_limit` zone (30 requests/second, burst=20, nodelay) for all `/api/` endpoints returning HTTP 429 when breached.
- **HTTP Security Headers in Nginx**:
  - `X-Frame-Options "DENY"`
  - `X-Content-Type-Options "nosniff"`
  - `Referrer-Policy "strict-origin-when-cross-origin"`
  - `Content-Security-Policy "default-src 'self' data: blob: 'unsafe-inline';"`
- **Go Storage Service Base Image Update**:
  - Update `services/storage-service/Dockerfile` base Golang image to latest patch release (`golang:1.25-alpine`) to resolve `govulncheck` standard library vulnerability warnings.
- **TDD Verification Tests**:
  - Unit/integration tests verifying HTTP Security Headers presence and Rate Limiting enforcement.

## Capabilities

### New Capabilities
- `security-hardening`: Defines HTTP security header enforcement, API rate limiting thresholds, and security base image patch requirements for SimpleCloud.

### Modified Capabilities

## Impact

- `services/web-frontend/nginx.conf`: Rate limiting zones and header rules added.
- `services/web-frontend/Dockerfile`: Unchanged structure, Nginx configuration reloaded.
- `services/storage-service/Dockerfile`: Updated base Golang image tag.
- `services/storage-service/internal/handler/web_test.go` or `nginx_security_test.go`: Integration tests added/updated to verify security headers and 429 Rate Limiting behavior.
