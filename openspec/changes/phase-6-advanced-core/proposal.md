## Why

SimpleCloud currently supports binary file uploads, user authentication via HTTP-only cookies, and a basic flat file grid/list web UI. However, as the system grows into a full-featured cloud storage solution:
1. Users lack visual folder organization and nested directory navigation.
2. Unauthenticated visitors or users with expired sessions encounter generic error toasts rather than a seamless login modal interface.
3. Storage quotas are stored in the database but not strictly enforced during file uploads (no HTTP 413 check).
4. Expired user sessions remain in PostgreSQL indefinitely without background cleanup.

Phase 6 Advanced Core solves these operational and user experience gaps by introducing nested folders, automatic UI auth recovery, strict quota enforcement, and background session garbage collection.

## What Changes

- **Frontend Login & Auth Modal UI**: Intercept HTTP `401 Unauthorized` responses in `app.js`. Display an interactive login modal overlay without clearing the current UI state. Re-authenticate and silently retry the failed request upon successful login.
- **Nested Folder Hierarchy & Path Browsing**:
  - Add `folders` PostgreSQL table (`id`, `user_id`, `parent_id`, `name`, `created_at`).
  - Add `folder_id` foreign key column to the `files` table (NULLable, referencing `folders(id)` ON DELETE CASCADE).
  - Add REST API endpoints: `POST /api/v1/folders`, `GET /api/v1/folders`, `DELETE /api/v1/folders/:id`.
  - Update `GET /api/v1/files` to accept `folder_id` query parameter for folder-scoped listing.
  - Update Web UI with folder card/row rendering, double-click navigation into folders, breadcrumb trail navigation, and folder creation modal.
- **Storage Quota Enforcement**:
  - Backend check during `POST /api/v1/files/upload`: reject upload with `413 Payload Too Large` if `user.used_bytes + incoming_file_size > user.quota_bytes`.
  - Update Web UI sidebar with a visual quota progress bar showing `used_bytes` / `quota_bytes` and percentage color indicators.
- **Automated Session Expiration Worker**:
  - Implement a background Go ticker worker in `storage-service` running every 1 minute to delete expired records (`expires_at < NOW()`) from the `user_sessions` table.

## Capabilities

### New Capabilities
- `folder-management`: Introduces folder CRUD operations, parent-child folder hierarchy, and folder-scoped file grouping.

### Modified Capabilities
- `web-frontend`: Updates `app.js` with 401 interceptor, login modal overlay, folder navigation views, breadcrumbs bar, and quota progress indicator.
- `file-storage`: Enforces quota boundaries (413 response on quota overflow) during file uploads and updates `used_bytes` aggregation.
- `auth-multi-tenancy`: Adds background ticker goroutine to automatically clean up expired user sessions from PostgreSQL `user_sessions`.

## Impact

- **Database**: New PostgreSQL migration `000003_folder_schema.sql` introducing `folders` table and adding `folder_id` column to `files`.
- **API Endpoints**:
  - `POST /api/v1/folders`
  - `GET /api/v1/folders`
  - `DELETE /api/v1/folders/:id`
  - `GET /api/v1/files` (extended with `folder_id` query param)
  - `POST /api/v1/files/upload` (returns 413 if quota exceeded)
- **Services**: `storage-service` (Go API handlers, background session cleaner), `web-frontend` (`app.js`, `index.html`, `styles.css`).
