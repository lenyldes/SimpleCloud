## Why

SimpleCloud currently has a basic project scaffold and health endpoint, but lacks persistent data storage and binary file management capabilities. Phase 2 introduces PostgreSQL 16 database storage for metadata and user quotas, paired with a high-performance disk storage engine featuring 2-level subfolder sharding and streaming uploads/downloads.

## What Changes

- Add PostgreSQL 16 container service to `docker-compose.yml` with health check and persistent volume.
- Create database schema migrations (`users` table for quotas, `files` table for file metadata, storage paths, and SHA256 hashes).
- Implement Go database connection pool and embedded SQL schema auto-migration runner.
- Implement disk storage engine with 2-level subfolder sharding (`/storage/f4/7a/<uuid>`) to prevent filesystem performance degradation.
- Implement streaming file upload (`POST /api/v1/files/upload`) with on-the-fly SHA256 hashing and immediate quota enforcement (aborting connection with 413 Payload Too Large if user quota is exceeded).
- Implement file download endpoint (`GET /api/v1/files/download/:id`) with proper headers (`Content-Disposition`, `Content-Type`, `Content-Length`) and file list endpoint (`GET /api/v1/files`).
- Provide unit and integration test suite following TDD principles.

## Capabilities

### New Capabilities
- `file-storage`: Binary file storage on disk with subfolder sharding, PostgreSQL metadata persistence, quota enforcement, and streaming upload/download API endpoints.

### Modified Capabilities

## Impact

- `docker-compose.yml`: Adds `postgres:16-alpine` service container and volume mappings.
- `services/storage-service`: Adds database connection pool, storage engine, streaming HTTP handlers, and embedded migration scripts.
- Runtime storage directory: `./data/storage` mapped to `/storage` in Docker container.
