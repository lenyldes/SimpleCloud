## MODIFIED Requirements

### Requirement: File Upload Streaming and Quota Enforcement
The system SHALL accept binary file uploads via `POST /api/v1/files/upload`, calculate the SHA256 hash on-the-fly during upload, and enforce user storage quota using a two-phase check. The upload filename SHALL NOT be empty and SHALL NOT exceed 255 characters; requests with invalid filenames SHALL be rejected immediately with HTTP status 400 Bad Request. File streaming and physical disk writing SHALL execute without holding an open database transaction. After the binary content is saved to disk, the system SHALL execute a short atomic database transaction (1-3 ms) that locks the user row, verifies remaining quota against the actual stored size, persists the file metadata record, and increments `used_bytes`. If the quota is exceeded or metadata persistence fails, the transaction SHALL be rolled back, the written binary file SHALL be removed from disk (with any deletion error logged), and an appropriate HTTP error (413 or 500) SHALL be returned.

#### Scenario: Successful file upload
- **WHEN** a user uploads a valid file within their available storage quota
- **THEN** the system streams the file to the sharded disk path `/storage/<hash1>/<hash2>/<uuid>` without holding a DB transaction, then commits metadata and increments `used_bytes` in a short transaction, returning HTTP status 201 Created with file metadata JSON.

#### Scenario: File upload exceeds user storage quota
- **WHEN** a user attempts to upload a file whose size exceeds their remaining storage quota (either before streaming or upon final atomic transaction verification)
- **THEN** the system aborts the upload or rolls back the transaction, cleans up any physical file written to disk, and returns HTTP status 413 Payload Too Large without modifying `used_bytes`.

#### Scenario: Metadata persistence failure rolls back disk write
- **WHEN** the binary file was written to disk but the metadata/quota database commit fails
- **THEN** the system rolls back the database transaction, removes the written binary file from disk with error logging if removal fails, returns HTTP 500, and leaves `used_bytes` unchanged.

#### Scenario: Metadata survives service restart
- **WHEN** the storage service is restarted after a successful upload
- **THEN** the uploaded file remains visible in the owner's file list and downloadable, because metadata is read from PostgreSQL and not from process memory.

#### Scenario: Filename exceeds 255 characters
- **WHEN** an authenticated user attempts to upload a file whose filename is longer than 255 characters
- **THEN** the system rejects the request with HTTP status 400 Bad Request and an error message indicating filename length limit exceeded, without writing files or modifying database records.

#### Scenario: Filename is empty
- **WHEN** an authenticated user attempts to upload a file with an empty filename
- **THEN** the system rejects the request with HTTP status 400 Bad Request.

### Requirement: Backend Quota Calculation and 413 Payload Check
The system SHALL check `users.used_bytes + incoming_content_length <= users.quota_bytes` optimistically before and during file upload streaming, comparing incoming size against the user's remaining quota (`quota_bytes - used_bytes`). The final authoritative quota check and the `used_bytes` increment SHALL execute within a short atomic database transaction that locks the user row (`SELECT ... FOR UPDATE`) after disk writing completes, guaranteeing that concurrent uploads cannot exceed the quota while preventing long-held database locks during network I/O.

#### Scenario: File upload exceeds quota boundary
- **WHEN** user uploads a file where `used_bytes + incoming_size > quota_bytes`
- **THEN** system SHALL abort streaming or roll back the final transaction, clean up any file created on disk, return `413 Payload Too Large`, and leave `used_bytes` unchanged.

#### Scenario: Multiple uploads exhaust remaining quota cumulatively
- **WHEN** a user with a 5 GB quota uploads files totaling 4 GB and then attempts a 2 GB upload
- **THEN** the system rejects the second upload with HTTP 413 because only ~1 GB of quota remains, and `used_bytes` reflects only the first uploads.

#### Scenario: Concurrent uploads cannot overdraw quota
- **WHEN** two concurrent uploads from the same user each fit the remaining quota alone but not together
- **THEN** the row-level locking during the final atomic commit ensures at most one succeeds, while the second sees an exhausted quota, rolls back, cleans up its written file, and returns HTTP 413.
