## MODIFIED Requirements

### Requirement: File Download and Metadata Listing
The system SHALL stream stored binary files via `GET /api/v1/files/download/:id` with appropriate HTTP headers (`Content-Disposition`, `Content-Type`, `Content-Length`), and return a JSON list of all uploaded files via `GET /api/v1/files`. Both operations SHALL resolve file metadata exclusively from PostgreSQL and enforce ownership: a file record that does not exist or belongs to another user SHALL be indistinguishable and answered with HTTP 404. The `:id` path segment SHALL be validated as a UUID before any lookup. The `Content-Disposition` header SHALL carry both an ASCII-safe fallback `filename` and an RFC 5987 `filename*` (`UTF-8''<percent-encoded>`) value so non-ASCII (e.g. Cyrillic) filenames download with correct names, and the header values SHALL be sanitized so filename characters (`\r`, `\n`, `"`) cannot inject additional headers.

#### Scenario: Successful file download
- **WHEN** a request is received from the owner for an existing file ID via `GET /api/v1/files/download/:id`
- **THEN** the system streams the binary file content with HTTP 200 OK and accurate header metadata.

#### Scenario: File not found download
- **WHEN** a request is received for a non-existent file ID via `GET /api/v1/files/download/:id`
- **THEN** the system returns HTTP 404 Not Found with a JSON error payload.

#### Scenario: Downloading another user's file is denied
- **WHEN** an authenticated user requests `GET /api/v1/files/download/:id` for a file owned by a different user
- **THEN** the system returns HTTP 404 Not Found identical to the non-existent file response, without revealing that the file exists.

#### Scenario: Malformed file ID download
- **WHEN** a request is received via `GET /api/v1/files/download/:id` where `:id` is not a valid UUID
- **THEN** the system returns a 4xx client error before any database or filesystem access.

#### Scenario: List user files
- **WHEN** a request is made to `GET /api/v1/files`
- **THEN** the system returns HTTP 200 OK with a JSON array containing only the requesting user's file metadata records read from PostgreSQL.

#### Scenario: UTF-8 filename carries RFC 5987 filename*
- **WHEN** the owner downloads a file whose stored name contains non-ASCII characters (e.g. `отчёт.pdf`)
- **THEN** the `Content-Disposition` header SHALL contain an ASCII-only fallback `filename` and a `filename*=UTF-8''<percent-encoded>` value that decodes to the original name.

#### Scenario: ASCII filename remains intact
- **WHEN** the owner downloads a file whose stored name is plain ASCII (e.g. `report.pdf`)
- **THEN** the `Content-Disposition` header SHALL contain `filename="report.pdf"` and the filename value SHALL equal the original name.

#### Scenario: Filename cannot inject headers
- **WHEN** a file's stored name contains CR/LF or double-quote characters
- **THEN** the `Content-Disposition` header SHALL be sanitized so no additional header lines or unescaped quotes are produced.
