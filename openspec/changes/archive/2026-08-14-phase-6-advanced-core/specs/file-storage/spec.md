## ADDED Requirements

### Requirement: Folder-Scoped Upload and Folder ID Binding
The system SHALL accept optional `folder_id` in form data during `POST /api/v1/files/upload`, binding uploaded files to the specified parent folder in PostgreSQL.

#### Scenario: Uploading file into a subfolder
- **WHEN** authenticated user uploads a file with `folder_id = <valid_folder_id>`
- **THEN** system SHALL verify folder ownership and save file record with `folder_id = <valid_folder_id>` in PostgreSQL.

### Requirement: Backend Quota Calculation and 413 Payload Check
The system SHALL check `users.used_bytes + incoming_content_length <= users.quota_bytes` before and during file upload streaming.

#### Scenario: File upload exceeds quota boundary
- **WHEN** user uploads a file where `used_bytes + incoming_size > quota_bytes`
- **THEN** system SHALL abort streaming immediately, clean up any temporary file created, return `413 Payload Too Large`, and leave `used_bytes` unchanged.
