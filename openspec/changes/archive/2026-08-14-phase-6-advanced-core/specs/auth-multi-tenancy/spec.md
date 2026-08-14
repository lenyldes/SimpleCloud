## ADDED Requirements

### Requirement: Session Expiration Background Garbage Collector
The system SHALL run a background Go ticker worker in `storage-service` (running at a configurable interval, default 1 minute) to automatically purge expired user sessions (`expires_at < NOW()`) from the `user_sessions` PostgreSQL table.

#### Scenario: Background session purging
- **WHEN** ticker interval elapses and there are sessions in `user_sessions` with `expires_at < CURRENT_TIMESTAMP`
- **THEN** background worker SHALL execute `DELETE FROM user_sessions WHERE expires_at < CURRENT_TIMESTAMP` and log the count of purged sessions.
