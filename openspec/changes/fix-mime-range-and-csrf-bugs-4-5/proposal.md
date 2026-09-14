## Why

Resolves two critical delivery and networking bugs identified during user acceptance testing (`BUGS.md`):
1. **BUG-4 (File Previews & Media Streaming)**: Currently, `GET /api/v1/files/download/:id` unconditionally forces `Content-Type: application/octet-stream` and streams files via plain `io.Copy`, which prevents browsers from rendering images and plain text in modal previews, and fails HTML5 video/audio playback due to lack of HTTP Range requests (`206 Partial Content`).
2. **BUG-5 (CSRF Port Mismatch)**: Nginx reverse proxy forwards `Host $host` and `X-Forwarded-Host $host`, which strips non-standard host ports (e.g. `:32214`). The Go CSRF middleware `RequireSameOrigin` compares the browser `Origin` header (containing the port) against `X-Forwarded-Host` (without port), resulting in `403 Forbidden` on login and mutating actions.

## What Changes

- **File Download MIME Detection & Range Requests (BUG-4)**:
  - Detect content MIME type dynamically via `mime.TypeByExtension(filepath.Ext(filename))` with fallback to `application/octet-stream`.
  - Replace manual header writing and `io.Copy` in `DownloadHandler` with Go's standard library `http.ServeContent(w, r, filename, stat.ModTime(), f)`.
  - Enable automatic support for HTTP Range requests (`Range: bytes=...`), returning `206 Partial Content`, `Content-Range`, `Accept-Ranges: bytes`, and caching headers (`If-Modified-Since` -> `304 Not Modified`).
  - Preserve `Content-Disposition` header with RFC 5987 UTF-8 filename encoding and ASCII fallback.
- **Nginx & Go CSRF Port Normalization (BUG-5)**:
  - In `services/web-frontend/nginx.conf`, change `proxy_set_header Host` and `proxy_set_header X-Forwarded-Host` from `$host` to `$http_host` for `/api/v1/auth/login` and `/api/` locations to preserve custom client ports.
  - In `RequireSameOrigin` middleware (`services/storage-service/internal/auth/handler.go`), improve host matching logic so that if the proxy forwarded host omits port, or if port matches, origin verification succeeds reliably.

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `file-storage`: Add MIME type detection and standard HTTP Range (`206 Partial Content`) streaming support to `Requirement: File Download and Metadata Listing`.
- `auth-multi-tenancy`: Clarify port preservation and tolerance in `Requirement: Cross-Site Request Forgery Origin Validation` for custom ports (e.g. reverse proxy mappings).

## Impact

- **Backend (`services/storage-service`)**:
  - `internal/handler/file_download.go`: MIME detection and `http.ServeContent`.
  - `internal/handler/file_download_test.go`: Unit and integration tests for MIME detection, Range requests (`206 Partial Content`), and RFC 5987 headers.
  - `internal/auth/handler.go`: Robust host/port comparison in `RequireSameOrigin`.
  - `internal/auth/handler_session_test.go`: Tests for custom port origins in CSRF middleware.
- **Infrastructure (`services/web-frontend`)**:
  - `nginx.conf`: Update `Host` and `X-Forwarded-Host` proxy headers to `$http_host`.
- **User Experience (`BUGS.md`)**:
  - Checklist item `2.5. Превью файлов` works for images, text, and media.
  - Form submissions and login work seamlessly when deployed on non-standard ports.
