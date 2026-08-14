## Context

SimpleCloud uses Nginx (`services/web-frontend`) as an HTTP reverse proxy in front of the Go backend (`services/storage-service`). See `proposal.md` for motivation.

## Goals / Non-Goals

**Goals:**
- Configure Nginx rate limiting zones (`limit_req_zone`) for `/api/v1/auth/login` (5r/s) and `/api/` (30r/s) returning `429 Too Many Requests`.
- Inject HTTP security headers (`X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Content-Security-Policy`) into all Nginx responses.
- Update `services/storage-service/Dockerfile` base image tag to patch standard library vulnerabilities.
- Add integration test suite (`nginx_security_test.go` or `web_test.go`) verifying header delivery and rate limiting.

**Non-Goals:**
- In-memory application level rate limiting inside Go backend (handled cleanly at reverse proxy tier).
- Complex application-wide logging overhaul in Go (deferred to future phase).

## Decisions

### Decision 1: Rate Limiting at Nginx layer rather than Go Middleware
- **Rationale**: Handling rate limiting at the Nginx layer drops excessive traffic before consuming Go goroutines or database connections, mitigating DDoS and bcrypt hash exhaustion attacks.
- **Alternatives Considered**: Go middleware using `golang.org/x/time/rate`. Rejected because Nginx is already the front gateway and handles client IPs natively with `binary_remote_addr`.

### Decision 2: Declarative HTTP Headers in Nginx
- **Rationale**: Setting `add_header` directives in `nginx.conf` ensures static assets and proxied API endpoints consistently output security headers without modifying Go handler code.

### Decision 3: Go Base Image Tag Upgrade
- **Rationale**: Upgrading `Dockerfile` builder and runtime stage to `golang:1.25-alpine` (or latest alpine base) ensures stdlib patches (`net/http`, `net/url`, `crypto/tls`) are incorporated.

## Risks / Trade-offs

- **[Risk]** Strict Rate Limiting might block legitimate users under high concurrency.
  - **Mitigation**: Configured `burst=5 nodelay` on login and `burst=20 nodelay` on API endpoints to absorb legitimate bursting.
