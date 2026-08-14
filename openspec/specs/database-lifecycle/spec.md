# Database Lifecycle Specification

## Purpose

Defines how the storage service builds its PostgreSQL connection string from environment configuration and how embedded SQL schema migrations are tracked and executed exactly once per version.

## Requirements

### Requirement: Escaped Database Connection String Construction
The system SHALL construct the PostgreSQL connection URL programmatically with proper URL-encoding of credentials, so passwords containing reserved characters (`@`, `:`, `/`, `%`, etc.) do not corrupt the connection string.

#### Scenario: Password with reserved characters
- **WHEN** the service starts with a `POSTGRES_PASSWORD` containing characters such as `@`, `:`, `/`, or `%`
- **THEN** the constructed connection URL SHALL percent-encode the password and the service SHALL connect successfully to the configured host, port, and database.

#### Scenario: Plain password unchanged
- **WHEN** the service starts with an alphanumeric password
- **THEN** the constructed connection URL SHALL point to the same configured host, port, database, and sslmode as before.

### Requirement: Versioned Migration Journal
The system SHALL record applied migrations in a `schema_migrations` journal table (`version`, `applied_at`). Each embedded migration file SHALL be executed at most once per database, only if its version is absent from the journal, and each migration SHALL run inside a transaction that applies the SQL and records the version atomically.

#### Scenario: Fresh database applies all migrations
- **WHEN** the service starts against a database without the journal table
- **THEN** the system SHALL create the journal and execute every embedded migration in order, recording one journal row per migration.

#### Scenario: Restart does not re-run migrations
- **WHEN** the service restarts against a database where all migrations are already recorded in the journal
- **THEN** the system SHALL execute zero migration bodies and start normally.

#### Scenario: Failed migration leaves no partial state
- **WHEN** a migration's SQL fails mid-execution
- **THEN** the transaction SHALL roll back so neither schema changes nor the journal entry persist, and the service SHALL fail to start with a descriptive error.

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

