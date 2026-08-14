## Purpose

Provides folder management capabilities including hierarchical parent-child directory creation, listing, path breadcrumb navigation, and cascading deletion of folders and contained files.

## ADDED Requirements

### Requirement: Folder Creation and Hierarchy Schema
The system SHALL store user directories in a `folders` PostgreSQL table (`id`, `user_id`, `parent_id`, `name`, `created_at`) and allow authenticated users to create nested subfolders via `POST /api/v1/folders`.

#### Scenario: Successful folder creation in root directory
- **WHEN** authenticated user sends `POST /api/v1/folders` with `{"name": "Documents"}` (no `parent_id`)
- **THEN** system SHALL create the folder with `parent_id = NULL` bound to user's `user_id` and return `201 Created` with folder JSON.

#### Scenario: Successful nested subfolder creation
- **WHEN** authenticated user sends `POST /api/v1/folders` with `{"name": "Work", "parent_id": "<valid_folder_id>"}`
- **THEN** system SHALL verify that parent folder exists and belongs to the user, insert the subfolder with `parent_id = <valid_folder_id>`, and return `201 Created`.

#### Scenario: Creating folder with parent belonging to another user
- **WHEN** User A sends `POST /api/v1/folders` with `parent_id` referencing User B's folder
- **THEN** system SHALL reject the request with `404 Not Found` or `403 Forbidden` without exposing User B's folder.

### Requirement: Folder-Scoped Directory Listing
The system SHALL support listing folders and files filtered by parent directory via `GET /api/v1/folders?parent_id=...` and `GET /api/v1/files?folder_id=...`.

#### Scenario: Listing root level folders and files
- **WHEN** authenticated user calls `GET /api/v1/folders` and `GET /api/v1/files` without parent/folder IDs
- **THEN** system SHALL return root-level folders (`parent_id IS NULL`) and root-level files (`folder_id IS NULL`).

#### Scenario: Listing contents inside a subfolder
- **WHEN** authenticated user calls `GET /api/v1/folders?parent_id=<folder_id>` and `GET /api/v1/files?folder_id=<folder_id>`
- **THEN** system SHALL return only subfolders and files directly nested within `<folder_id>`.

### Requirement: Recursive Folder Deletion
The system SHALL support folder deletion via `DELETE /api/v1/folders/:id`, recursively unlinking and deleting all subfolders, contained files from PostgreSQL, and removing associated binary file shards from physical disk storage.

#### Scenario: Deleting non-empty folder
- **WHEN** authenticated user sends `DELETE /api/v1/folders/<folder_id>`
- **THEN** system SHALL locate all child subfolders and files recursively, remove physical binary file shards from disk, decrement the user's `used_bytes` quota accordingly, delete DB records, and return `200 OK`.
