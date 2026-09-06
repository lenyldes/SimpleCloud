## MODIFIED Requirements

### Requirement: Subfolder Storage Sharding
The system SHALL store uploaded binary files on disk using a 2-level subfolder sharding structure based on the file UUID (e.g., `/storage/<uuid[0..1]>/<uuid[2..3]>/<uuid>`). Parent shard subdirectories SHALL be created with mode `0775`, and saved binary files SHALL have permissions `0664` so they are accessible and readable directly by the host user. The sharded path builder SHALL reject identifiers that are not plain filenames (containing path separators or resolving outside the base directory) as defense-in-depth against path traversal, independent of HTTP-layer normalization.

#### Scenario: Subfolder sharded file creation
- **WHEN** a new file with UUID `f47a8b90-1234-5678-9abc-def012345678` is stored on disk
- **THEN** the system automatically creates parent directories with mode `0775` if needed and saves the file at `/storage/f4/7a/f47a8b90-1234-5678-9abc-def012345678` with permissions `0664`.

#### Scenario: Sharded path rejects traversal identifiers
- **WHEN** the sharded path builder receives an identifier containing `/`, `\`, or whose base-name differs from the identifier itself
- **THEN** it SHALL return an invalid-identifier error and MUST NOT return a path.

## ADDED Requirements

### Requirement: Storage Directory Initialization and Permissions Check
The storage service SHALL verify on startup that the configured base storage directory exists and is writable. If the directory does not exist, the system SHALL attempt to create it with mode `0775`. The system SHALL verify write access using a probe file creation and removal check; if write access is denied or directory creation fails, the service startup SHALL fail fast with an explanatory fatal error.

#### Scenario: Successful storage directory initialization
- **WHEN** the storage service starts with a valid and writable storage directory
- **THEN** the startup check creates the directory if absent, confirms write access, and allows the service to begin serving requests.

#### Scenario: Unwritable storage directory causes fail-fast shutdown
- **WHEN** the storage service starts with a storage directory that cannot be created or is read-only
- **THEN** the service logs a clear diagnostic error indicating the permission failure and halts startup before binding network listeners.
