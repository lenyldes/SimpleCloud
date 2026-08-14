# file-storage delta — phase7-hardening-and-polish

## ADDED Requirements

### Requirement: Resource ID UUID Validation
All file and folder endpoints that accept an identifier in the URL path (file download, file deletion, folder deletion) SHALL parse the identifier as a UUID before any filesystem or database access. Requests with malformed identifiers SHALL be rejected with a 4xx client error without touching the filesystem.

#### Scenario: Download with non-UUID identifier
- **WHEN** an authenticated client requests `GET /api/v1/files/download/<not-a-uuid>`
- **THEN** the system SHALL respond with `400 Bad Request` or `404 Not Found` and MUST NOT build any filesystem path from the identifier.

#### Scenario: Deletion with non-UUID identifier
- **WHEN** an authenticated client requests deletion of a file or folder using a non-UUID path segment
- **THEN** the system SHALL respond with a 4xx client error and MUST NOT delete or inspect any resource.

#### Scenario: Traversal-looking identifier is rejected
- **WHEN** an identifier contains path separators or traversal sequences (e.g. `..`, `/`, `\`)
- **THEN** UUID parsing SHALL fail and the request SHALL be rejected before any path construction.

## MODIFIED Requirements

### Requirement: Subfolder Storage Sharding
The system SHALL store uploaded binary files on disk using a 2-level subfolder sharding structure based on the file UUID (e.g., `/storage/<uuid[0..1]>/<uuid[2..3]>/<uuid>`). The sharded path builder SHALL reject identifiers that are not plain filenames (containing path separators or resolving outside the base directory) as defense-in-depth against path traversal, independent of HTTP-layer normalization.

#### Scenario: Subfolder sharded file creation
- **WHEN** a new file with UUID `f47a8b90-1234-5678-9abc-def012345678` is stored on disk
- **THEN** the system automatically creates parent directories if needed and saves the file at `/storage/f4/7a/f47a8b90-1234-5678-9abc-def012345678`.

#### Scenario: Sharded path rejects traversal identifiers
- **WHEN** the sharded path builder receives an identifier containing `/`, `\`, or whose base-name differs from the identifier itself
- **THEN** it SHALL return an invalid-identifier error and MUST NOT return a path.

### Requirement: File Download and Metadata Listing
The system SHALL stream stored binary files via `GET /api/v1/files/download/:id` with appropriate HTTP headers (`Content-Disposition`, `Content-Type`, `Content-Length`), and return a JSON list of all uploaded files via `GET /api/v1/files`. Both operations SHALL resolve file metadata exclusively from PostgreSQL and enforce ownership: a file record that does not exist or belongs to another user SHALL be indistinguishable and answered with HTTP 404. The `:id` path segment SHALL be validated as a UUID before any lookup.

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
