## MODIFIED Requirements

### Requirement: File Upload Streaming and Quota Enforcement
The system SHALL accept binary file uploads via `POST /api/v1/files/upload`, calculate the SHA256 hash on-the-fly during upload, and check the user's available storage quota continuously. If the file upload exceeds the user's available quota, the system SHALL immediately abort the connection with HTTP status 413 Payload Too Large and delete any partially written temporary file from disk. After the binary content is stored, the system SHALL persist the file metadata record and increment the user's `used_bytes` atomically in PostgreSQL; if metadata persistence fails, the system SHALL delete the already written binary file from disk and return HTTP 500 instead of leaving an ownerless file.

#### Scenario: Successful file upload
- **WHEN** a user uploads a valid file within their available storage quota
- **THEN** the system streams the file to the sharded disk path `/storage/<hash1>/<hash2>/<uuid>`, saves file metadata and SHA256 in PostgreSQL, updates the user's used storage bytes, and returns HTTP status 201 Created with file metadata JSON.

#### Scenario: File upload exceeds user storage quota
- **WHEN** a user attempts to upload a file whose size exceeds their remaining storage quota
- **THEN** the system immediately aborts the upload stream with HTTP status 413 Payload Too Large, cleans up any temporary file fragment from disk, and leaves the user's storage quota unchanged in PostgreSQL.

#### Scenario: Metadata persistence failure rolls back disk write
- **WHEN** the binary file was written to disk but the metadata/quota database commit fails
- **THEN** the system deletes the written binary file from disk, returns HTTP 500, and leaves `used_bytes` unchanged.

#### Scenario: Metadata survives service restart
- **WHEN** the storage service is restarted after a successful upload
- **THEN** the uploaded file remains visible in the owner's file list and downloadable, because metadata is read from PostgreSQL and not from process memory.

### Requirement: File Download and Metadata Listing
The system SHALL stream stored binary files via `GET /api/v1/files/download/:id` with appropriate HTTP headers (`Content-Disposition`, `Content-Type`, `Content-Length`), and return a JSON list of all uploaded files via `GET /api/v1/files`. Both operations SHALL resolve file metadata exclusively from PostgreSQL and enforce ownership: a file record that does not exist or belongs to another user SHALL be indistinguishable and answered with HTTP 404.

#### Scenario: Successful file download
- **WHEN** a request is received from the owner for an existing file ID via `GET /api/v1/files/download/:id`
- **THEN** the system streams the binary file content with HTTP 200 OK and accurate header metadata.

#### Scenario: File not found download
- **WHEN** a request is received for a non-existent file ID via `GET /api/v1/files/download/:id`
- **THEN** the system returns HTTP 404 Not Found with a JSON error payload.

#### Scenario: Downloading another user's file is denied
- **WHEN** an authenticated user requests `GET /api/v1/files/download/:id` for a file owned by a different user
- **THEN** the system returns HTTP 404 Not Found identical to the non-existent file response, without revealing that the file exists.

#### Scenario: List user files
- **WHEN** a request is made to `GET /api/v1/files`
- **THEN** the system returns HTTP 200 OK with a JSON array containing only the requesting user's file metadata records read from PostgreSQL.

### Requirement: Backend Quota Calculation and 413 Payload Check
The system SHALL check `users.used_bytes + incoming_content_length <= users.quota_bytes` before and during file upload streaming, comparing the incoming size against the user's REMAINING quota (`quota_bytes - used_bytes`) rather than the total quota. The quota check and the `used_bytes` increment SHALL execute within the same database transaction that locks the user row against concurrent updates.

#### Scenario: File upload exceeds quota boundary
- **WHEN** user uploads a file where `used_bytes + incoming_size > quota_bytes`
- **THEN** system SHALL abort streaming immediately, clean up any temporary file created, return `413 Payload Too Large`, and leave `used_bytes` unchanged.

#### Scenario: Multiple uploads exhaust remaining quota cumulatively
- **WHEN** a user with a 5 GB quota uploads files totaling 4 GB and then attempts a 2 GB upload
- **THEN** the system rejects the second upload with HTTP 413 because only ~1 GB of quota remains, and `used_bytes` reflects only the first uploads.

#### Scenario: Concurrent uploads cannot overdraw quota
- **WHEN** two concurrent uploads from the same user each fit the remaining quota alone but not together
- **THEN** the user row locking ensures at most one succeeds and the sum of persisted `used_bytes` never exceeds `quota_bytes`.

## ADDED Requirements

### Requirement: File Deletion and Quota Release
The system SHALL support deleting a single file via `DELETE /api/v1/files/:id` for authenticated users. The system SHALL verify ownership via PostgreSQL (records of other users or non-existent records return HTTP 404), remove the binary file from physical disk storage, delete the metadata record, and decrement the user's `used_bytes` without ever letting it become negative.

#### Scenario: Successful file deletion
- **WHEN** the owner sends `DELETE /api/v1/files/<file_id>` for an existing file
- **THEN** the system removes the binary file from disk, deletes the PostgreSQL record, decrements `used_bytes` by the file's `size_bytes`, and returns HTTP 200 OK.

#### Scenario: Deleting another user's file is denied
- **WHEN** an authenticated user sends `DELETE /api/v1/files/<file_id>` for a file owned by a different user
- **THEN** the system returns HTTP 404 Not Found and leaves both the binary file and the metadata record untouched.

#### Scenario: Deleting a non-existent file
- **WHEN** an authenticated user sends `DELETE /api/v1/files/<file_id>` with an unknown or malformed UUID
- **THEN** the system returns HTTP 404 Not Found with a JSON error payload.

#### Scenario: Quota accounting never goes negative
- **WHEN** a file is deleted whose recorded size exceeds the current `used_bytes` (inconsistent state)
- **THEN** the system clamps `used_bytes` at zero instead of storing a negative value.
