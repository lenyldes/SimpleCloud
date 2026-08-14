# project-scaffold delta — phase7-hardening-and-polish

## ADDED Requirements

### Requirement: Graceful Service Shutdown
The storage service SHALL handle termination signals (`SIGTERM`, `SIGINT`) by gracefully shutting down the HTTP server within a bounded timeout (allowing in-flight requests, including uploads, to finish), then closing the database connection pool before exiting.

#### Scenario: SIGTERM during active upload
- **WHEN** the service receives `SIGTERM` while an upload request is in progress
- **THEN** the service SHALL stop accepting new connections, allow the in-flight request to complete (within the shutdown timeout), close the database pool, and exit cleanly with a zero/success exit status.

#### Scenario: Shutdown timeout expires
- **WHEN** in-flight requests do not finish within the configured shutdown timeout
- **THEN** the service SHALL force shutdown and exit rather than hanging indefinitely.

#### Scenario: Startup failure surfaces immediately
- **WHEN** the HTTP server fails to start (e.g. port already in use)
- **THEN** the service SHALL log the error and terminate with a non-zero exit status.
