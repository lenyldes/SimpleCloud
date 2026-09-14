## Context

See `proposal.md` for motivation and background.

Currently in `services/storage-service/internal/handler/file_download.go`:
- `DownloadHandler` hardcodes `w.Header().Set("Content-Type", "application/octet-stream")`.
- Binary content is served via `io.Copy(w, f)` after `w.WriteHeader(http.StatusOK)`.
- Range headers (`Range: bytes=...`) sent by media players or download managers are ignored, preventing media seeking and failing preview in web modals.

In `services/web-frontend/nginx.conf`:
- Reverse proxy directives use `proxy_set_header Host $host;` and `proxy_set_header X-Forwarded-Host $host;`.
- In Nginx, `$host` strips non-standard port numbers (e.g., `localhost:32214` becomes `localhost`), causing CSRF check mismatch in `RequireSameOrigin`.

## Goals / Non-Goals

**Goals:**
- Dynamically detect MIME type from file extension using Go's standard library `mime.TypeByExtension(filepath.Ext(filename))`, with fallback to `application/octet-stream`.
- Delegate streaming to `http.ServeContent(w, r, filename, stat.ModTime(), f)`, which natively supports HTTP Range requests (`206 Partial Content`), `Accept-Ranges: bytes`, `Content-Range`, and conditional headers (`If-Modified-Since`, `If-Range`).
- Retain existing RFC 5987 UTF-8 `Content-Disposition` headers and sanitization.
- Forward full client host and port via `$http_host` in Nginx reverse proxy.
- Ensure `RequireSameOrigin` middleware handles custom ports and port omission reliably.

**Non-Goals:**
- Refactoring the upload pipeline or database transactions (reserved for `BUG-10`).
- Modifying UI modal dialogs or routing logic (reserved for `BUG-7` and `BUG-11`).

## Decisions

### Decision 1: Standard Library `http.ServeContent` over custom range parser
- **Rationale**: Go's `net/http` package provides `http.ServeContent`, which implements the full RFC 7233 HTTP Range specification, handles boundary parsing, multipart/byteranges if requested, `416 Range Not Satisfiable`, and caching headers. `*os.File` already implements `io.ReadSeeker`.
- **Alternatives considered**: Writing a manual byte-range parser with `f.Seek`. Rejected because it is complex, prone to off-by-one errors, and reinvents standard library functionality.

### Decision 2: Extension-based MIME detection before `http.ServeContent`
- **Rationale**: `http.ServeContent` sniffs the first 512 bytes if `Content-Type` is omitted, which can misclassify text/plain, SVGs, or custom media formats. By calling `mime.TypeByExtension(filepath.Ext(filename))` and setting `w.Header().Set("Content-Type", ...)` beforehand, we provide accurate content typing while allowing fallback to `application/octet-stream`.
- **Alternatives considered**: Sniffing bytes only via `http.DetectContentType`. Rejected because extension-based matching is standard for web asset delivery and handles CSS/JS/SVG/JSON more reliably.

### Decision 3: Nginx `$http_host` for host preservation
- **Rationale**: In Nginx, `$http_host` contains the exact value of the client's `Host` header, including the port number when present (e.g. `localhost:32214`). Setting `proxy_set_header Host $http_host;` and `proxy_set_header X-Forwarded-Host $http_host;` ensures the backend sees the exact origin requested by the browser.
- **Alternatives considered**: `$host:$server_port`. Rejected because in Docker container setups, `$server_port` inside the container (80) differs from the host port mapped on Docker (e.g. 32214). `$http_host` preserves what the browser actually sent.

### Decision 4: Go `RequireSameOrigin` defense-in-depth port comparison
- **Rationale**: In `RequireSameOrigin`, if `expectedHost` has no port (e.g. standard ports 80/443 or reverse proxy stripped it), but `u.Host` has a port, comparing `u.Hostname()` to `expectedHost` when `!strings.Contains(expectedHost, ":")` ensures legitimate requests pass without weakening CSRF security against third-party domains.

## Risks / Trade-offs

- **[Risk]** `http.ServeContent` overriding `Content-Disposition`.
  - **Mitigation**: In Go `net/http`, `http.ServeContent` only sets `Content-Type`, `Content-Length`, and `Last-Modified` if they are not already set. It preserves any existing `Content-Disposition` header set on `w.Header()`.
- **[Risk]** Missing MIME types on minimal Docker container environments (missing `/etc/mime.types`).
  - **Mitigation**: Go standard library initializes common MIME types built-in, and fallback to `application/octet-stream` ensures no blank headers. Common web extensions (`.png`, `.jpg`, `.txt`, `.pdf`, `.mp4`, `.webm`, `.json`) are always recognized.
