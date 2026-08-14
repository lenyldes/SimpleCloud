## 1. Test Agent Tasks (RED State)

- [ ] 1.1 Add automated integration tests for Nginx security headers (`X-Frame-Options`, `Content-Security-Policy`, etc.) in `web_test.go`
- [ ] 1.2 Add automated integration tests for Nginx rate limiting response (`HTTP 429`) under rapid requests in `web_test.go`

## 2. Code Implementation Tasks (GREEN State)

- [ ] 2.1 Update `services/web-frontend/nginx.conf` to add HTTP security headers (`X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Content-Security-Policy`)
- [ ] 2.2 Update `services/web-frontend/nginx.conf` to define `limit_req_zone` for `/api/v1/auth/login` (5r/s) and `/api/` (30r/s) returning 429
- [ ] 2.3 Update `services/storage-service/Dockerfile` base Golang image tag to resolve `govulncheck` vulnerabilities
- [ ] 2.4 Run `go test -v -cover ./...` and `govulncheck ./...` to verify all tests pass and coverage threshold (>= 85%) is maintained
