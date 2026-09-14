## MODIFIED Requirements

### Requirement: Folder Creation and Hierarchy Schema
The system SHALL store user directories in a `folders` PostgreSQL table (`id`, `user_id`, `parent_id`, `name`, `created_at`) and allow authenticated users to create nested subfolders via `POST /api/v1/folders`. Folder metadata SHALL be read from and written to PostgreSQL only; no in-memory folder registry exists. Folder names SHALL NOT be empty, SHALL NOT exceed 255 characters, and SHALL NOT contain path separators (`/`, `\`), traversal sequences (`..`), or null bytes (`\x00`). Requests violating folder name constraints SHALL be rejected with HTTP 400 Bad Request. If `parent_id` is provided but is not a valid UUID, the system SHALL reject the request with HTTP 400 instead of silently storing a NULL parent.

#### Scenario: Successful folder creation in root directory
- **WHEN** authenticated user sends `POST /api/v1/folders` with `{"name": "Documents"}` (no `parent_id`)
- **THEN** system SHALL create the folder with `parent_id = NULL` bound to user's `user_id` and return `201 Created` with folder JSON.

#### Scenario: Successful nested subfolder creation
- **WHEN** authenticated user sends `POST /api/v1/folders` with `{"name": "Work", "parent_id": "<valid_folder_id>"}`
- **THEN** system SHALL verify via PostgreSQL that parent folder exists and belongs to the user, insert the subfolder with `parent_id = <valid_folder_id>`, and return `201 Created`.

#### Scenario: Creating folder with parent belonging to another user
- **WHEN** User A sends `POST /api/v1/folders` with `parent_id` referencing User B's folder
- **THEN** system SHALL reject the request with `404 Not Found` or `403 Forbidden` without exposing User B's folder.

#### Scenario: Creating folder with invalid parent_id UUID
- **WHEN** authenticated user sends `POST /api/v1/folders` with a non-empty `parent_id` that does not parse as a UUID
- **THEN** system SHALL reject the request with HTTP 400 and an `invalid parent_id` error, without creating any folder record.

#### Scenario: Folder name contains null byte
- **WHEN** authenticated user sends `POST /api/v1/folders` with a name containing `\x00` (null byte)
- **THEN** system SHALL reject the request with HTTP 400 Bad Request without inserting any record into PostgreSQL.

#### Scenario: Folder metadata survives service restart
- **WHEN** the storage service is restarted after folder creation
- **THEN** the created folders remain visible in the owner's folder listing because metadata is read from PostgreSQL.
