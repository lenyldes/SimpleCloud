# File Storage Specification

## Purpose

Provides core binary file storage on disk with subfolder sharding, PostgreSQL metadata persistence, storage quota checking, and streaming upload/download HTTP endpoints.

## Requirements

### Requirement: File Upload Streaming and Quota Enforcement
The system SHALL accept binary file uploads via `POST /api/v1/files/upload`, calculate the SHA256 hash on-the-fly during upload, and check the user's available storage quota continuously. If the file upload exceeds the user's available quota, the system SHALL immediately abort the connection with HTTP status 413 Payload Too Large and delete any partially written temporary file from disk.

#### Scenario: Successful file upload
- **WHEN** a user uploads a valid file within their available storage quota
- **THEN** the system streams the file to the sharded disk path `/storage/<hash1>/<hash2>/<uuid>`, saves file metadata and SHA256 in PostgreSQL, updates the user's used storage bytes, and returns HTTP status 201 Created with file metadata JSON.

#### Scenario: File upload exceeds user storage quota
- **WHEN** a user attempts to upload a file whose size exceeds their remaining storage quota
- **THEN** the system immediately aborts the upload stream with HTTP status 413 Payload Too Large, cleans up any temporary file fragment from disk, and leaves the user's storage quota unchanged in PostgreSQL.

### Requirement: Subfolder Storage Sharding
The system SHALL store uploaded binary files on disk using a 2-level subfolder sharding structure based on the file UUID (e.g., `/storage/<uuid[0..1]>/<uuid[2..3]>/<uuid>`).

#### Scenario: Subfolder sharded file creation
- **WHEN** a new file with UUID `f47a8b90-1234-5678-9abc-def012345678` is stored on disk
- **THEN** the system automatically creates parent directories if needed and saves the file at `/storage/f4/7a/f47a8b90-1234-5678-9abc-def012345678`.

### Requirement: File Download and Metadata Listing
The system SHALL stream stored binary files via `GET /api/v1/files/download/:id` with appropriate HTTP headers (`Content-Disposition`, `Content-Type`, `Content-Length`), and return a JSON list of all uploaded files via `GET /api/v1/files`.

#### Scenario: Successful file download
- **WHEN** a request is received for an existing file ID via `GET /api/v1/files/download/:id`
- **THEN** the system streams the binary file content with HTTP 200 OK and accurate header metadata.

#### Scenario: File not found download
- **WHEN** a request is received for a non-existent file ID via `GET /api/v1/files/download/:id`
- **THEN** the system returns HTTP 404 Not Found with a JSON error payload.

#### Scenario: List user files
- **WHEN** a request is made to `GET /api/v1/files`
- **THEN** the system returns HTTP 200 OK with a JSON array containing file metadata records.
