# Folder Management Specification

## Purpose

Provides folder management capabilities including hierarchical parent-child directory creation, listing, path breadcrumb navigation, and cascading deletion of folders and contained files.

## Requirements

### Requirement: Folder Creation and Hierarchy Schema
The system SHALL store user directories in a `folders` PostgreSQL table (`id`, `user_id`, `parent_id`, `name`, `created_at`) and allow authenticated users to create nested subfolders via `POST /api/v1/folders`. Folder metadata SHALL be read from and written to PostgreSQL only; no in-memory folder registry exists. If `parent_id` is provided but is not a valid UUID, the system SHALL reject the request with HTTP 400 instead of silently storing a NULL parent.

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

#### Scenario: Folder metadata survives service restart
- **WHEN** the storage service is restarted after folder creation
- **THEN** the created folders remain visible in the owner's folder listing because metadata is read from PostgreSQL.

### Requirement: Folder-Scoped Directory Listing
The system SHALL support listing folders and files filtered by parent directory via `GET /api/v1/folders?parent_id=...` and `GET /api/v1/files?folder_id=...`, resolving all records exclusively from PostgreSQL and returning only records belonging to the requesting user.

#### Scenario: Listing root level folders and files
- **WHEN** authenticated user calls `GET /api/v1/folders` and `GET /api/v1/files` without parent/folder IDs
- **THEN** the system SHALL return root-level folders (`parent_id IS NULL`) and root-level files (`folder_id IS NULL`).

#### Scenario: Listing contents inside a subfolder
- **WHEN** authenticated user calls `GET /api/v1/folders?parent_id=<folder_id>` and `GET /api/v1/files?folder_id=<folder_id>`
- **THEN** the system SHALL return only subfolders and files directly nested within `<folder_id>` that belong to the requesting user.

### Requirement: Recursive Folder Deletion
The system SHALL support folder deletion via `DELETE /api/v1/folders/:id`, recursively unlinking and deleting all subfolders, contained files from PostgreSQL, and removing associated binary file shards from physical disk storage. Ownership SHALL be verified in PostgreSQL (unknown or foreign folders return HTTP 404). The recursive collection of descendant folders and files, the deletion of their database records, and the `used_bytes` decrement SHALL execute within a single database transaction; physical disk removal of the collected files SHALL happen as part of the same operation, and files whose disk removal fails SHALL NOT leave the operation hanging or inconsistent in the database.

#### Scenario: Deleting non-empty folder
- **WHEN** authenticated user sends `DELETE /api/v1/folders/<folder_id>`
- **THEN** system SHALL locate all child subfolders and files recursively, remove physical binary file shards from disk, decrement the user's `used_bytes` quota accordingly, delete DB records, and return `200 OK`.

#### Scenario: Deleting another user's folder is denied
- **WHEN** an authenticated user sends `DELETE /api/v1/folders/<folder_id>` for a folder owned by a different user
- **THEN** the system returns HTTP 404 Not Found and leaves the folder tree, files, and quota untouched.

#### Scenario: Files of a deleted folder are no longer downloadable
- **WHEN** a folder containing files has been deleted
- **THEN** subsequent `GET /api/v1/files/download/<deleted_file_id>` requests return HTTP 404 and the binary shards are absent from disk.
