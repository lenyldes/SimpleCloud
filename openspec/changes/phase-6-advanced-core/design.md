## Context

See `proposal.md` for motivation and scope. SimpleCloud uses Go (`storage-service`) with PostgreSQL 16 for backend API, and Vanilla HTML/CSS/JS (`web-frontend`) served via Nginx.

## Goals / Non-Goals

**Goals:**
- Implement clean database migrations for `folders` table and `files.folder_id` foreign key.
- Provide full RESTful CRUD API handlers for folders (`internal/handler/folder.go`).
- Enforce strict `user_id` multi-tenant scoping on all folder queries (`WHERE user_id = $1`).
- Enforce strict storage quotas (`used_bytes + incoming_file_size <= quota_bytes`) returning `413 Payload Too Large`.
- Provide a background ticker goroutine in `storage-service` to delete expired sessions (`expires_at < NOW()`).
- Implement UI 401 interceptor in `app.js` with an interactive auth modal overlay.
- Maintain **85%+ statement test coverage** across all new backend Go packages.

**Non-Goals:**
- In-browser text editing (Notepad) — deferred to Phase 8.
- Public file sharing links — deferred to Phase 8.
- Automated file/folder expiration worker — deferred to Phase 8.

## Decisions

### 1. Database Schema & Parent-Child Folder Modeling
- **Decision**: Create a `folders` table with self-referencing `parent_id` (UUID, NULLable) referencing `folders(id)` ON DELETE CASCADE. Add `folder_id` (UUID, NULLable) column to `files` table referencing `folders(id)` ON DELETE CASCADE.
- **Rationale**: Relational PostgreSQL foreign keys with `ON DELETE CASCADE` ensure database-level consistency when a folder is deleted, automatically cascading deletions to nested subfolders and files.
- **Alternatives Considered**:
  - *Path string column (e.g. `/docs/work/`)*: Harder to rename directories and maintain clean SQL indexing. Relational `parent_id` with UUIDs is far cleaner and more performant.

### 2. Recursive Folder Cleanup in Storage Service
- **Decision**: Before deleting a folder record from PostgreSQL, `DeleteFolder` handler retrieves all descendant files and subfolders using a recursive SQL CTE (`WITH RECURSIVE folder_tree AS ...`), deletes physical binary files from disk sharding, decrements `used_bytes` from `users`, and then executes `DELETE FROM folders WHERE id = $1`.
- **Rationale**: Prevents orphaned binary files on disk and maintains accurate quota accounting.

### 3. UI 401 Interception Architecture
- **Decision**: Wrap `fetch` calls in an `apiClient` helper in `app.js`. If a request receives HTTP 401, pause execution, remove `.hidden` from `#modal-auth`, and prompt for credentials. Upon successful login, store user info, close modal, and retry the pending API request.
- **Rationale**: Eliminates page refreshes and abrupt error toasts when sessions expire, providing a seamless user experience.

### 4. Background Session Garbage Collection
- **Decision**: Start a `time.NewTicker(1 * time.Minute)` goroutine inside `internal/auth/service.go` upon `storage-service` initialization. Worker calls `DB.Exec("DELETE FROM user_sessions WHERE expires_at < CURRENT_TIMESTAMP")`.
- **Rationale**: Keeps `user_sessions` table clean without requiring external cron jobs or extra Docker containers.

## Risks / Trade-offs

- **[Risk]** Large nested folder tree deletion causing long database transaction locks.
  - **Mitigation**: Fetch file paths to delete first, perform physical disk unlinking async or in batch, and execute DB deletion within a fast transaction.
- **[Risk]** Multiple 401 responses triggering overlapping Auth Modals.
  - **Mitigation**: Use a boolean flag `isAuthModalOpen` in `app.js` to ensure only one login modal is active at any time.

## Migration Plan

1. Execute migration `000003_folder_schema.sql` via Go `golang-migrate` on startup.
2. Deploy updated `storage-service` Docker container.
3. Deploy updated `web-frontend` Docker container with new Nginx assets.
