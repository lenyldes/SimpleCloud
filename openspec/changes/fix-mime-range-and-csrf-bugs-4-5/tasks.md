## 1. Test Suite Coverage (RED State - Test Agent)

- [ ] 1.1 Add unit tests in `file_download_test.go` for dynamic MIME type detection on download (verifying `.png`, `.jpg`, `.mp4`, `.txt`, `.json`, and unmapped fallback `application/octet-stream`)
- [ ] 1.2 Add unit tests in `file_download_test.go` for HTTP Range requests (`Range: bytes=0-10`), asserting `206 Partial Content`, `Content-Range: bytes 0-10/<total>`, `Accept-Ranges: bytes`, and exact byte slice payload
- [ ] 1.3 Add unit tests in `internal/auth/handler_session_test.go` for `RequireSameOrigin` validating mutating requests with custom ports (e.g. `Origin: http://localhost:32214` matching `X-Forwarded-Host: localhost:32214`)

## 2. Core Implementation (GREEN State - Code Agent)

- [ ] 2.1 Update `services/web-frontend/nginx.conf` to set `proxy_set_header Host $http_host;` and `proxy_set_header X-Forwarded-Host $http_host;` across `/api/v1/auth/login` and `/api/` locations
- [ ] 2.2 Update `RequireSameOrigin` in `services/storage-service/internal/auth/handler.go` with robust host and port comparison logic
- [ ] 2.3 Update `DownloadHandler` in `services/storage-service/internal/handler/file_download.go` to detect MIME type via `mime.TypeByExtension` and stream via `http.ServeContent` preserving RFC 5987 `Content-Disposition`
- [ ] 2.4 Verify all tests pass locally (`go test -v -cover ./...`), verify >= 85% statement coverage across `internal/*`, verify code formatting with `gofmt -l .`, and ensure line count rules (<= 450 lines) are strictly respected
