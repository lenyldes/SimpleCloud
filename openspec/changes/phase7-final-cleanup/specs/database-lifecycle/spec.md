## ADDED Requirements

### Requirement: Configurable Migration Source with Descriptive Failures
The migration runner SHALL execute SQL migrations from a configurable file source (defaulting to the embedded migrations), so that migration handling can be exercised with synthetic sources in tests. When the migration source is unusable, the system SHALL fail with a descriptive error identifying the cause instead of silently skipping migrations.

#### Scenario: Missing migrations directory
- **WHEN** migrations are run from a source that does not contain the `migrations` directory
- **THEN** the system SHALL return an error describing the unreadable migrations directory and apply nothing.

#### Scenario: Unreadable migration file
- **WHEN** a migration file listed in the source cannot be read
- **THEN** the system SHALL return an error naming the failing migration file and leave the database unchanged.

#### Scenario: Non-file entries are skipped
- **WHEN** the migrations directory contains subdirectories alongside SQL files
- **THEN** the system SHALL skip the subdirectories and execute only regular migration files.

#### Scenario: Empty migrations directory
- **WHEN** the migrations directory contains no migration files
- **THEN** the system SHALL complete successfully having applied nothing beyond the journal bootstrap.
