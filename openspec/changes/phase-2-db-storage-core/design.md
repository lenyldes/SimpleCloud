## Context

See `proposal.md` for motivation and scope. SimpleCloud is expanding from a basic Go HTTP health server to include PostgreSQL 16 metadata storage and a sharded binary disk storage engine.

## Goals / Non-Goals

**Goals:**
- Provide reliable PostgreSQL database storage for user quotas and file metadata.
- Implement embedded SQL schema migrations on Go service startup.
- Implement 2-level subfolder disk sharded storage engine (`/storage/f4/7a/<uuid>`).
- Implement streaming file uploads with on-the-fly SHA256 hashing and immediate quota checking with connection abort on quota limit breach.
- Provide HTTP API endpoints: `POST /api/v1/files/upload`, `GET /api/v1/files/download/:id`, `GET /api/v1/files`.

**Non-Goals:**
- User authentication and multi-tenancy auth headers (deferred to Phase 3; initial user record will be seeded/hardcoded for dev/tests).
- Web frontend UI (deferred to Phase 4).
- Automatic background file expiration worker execution (deferred to Phase 6).

## Decisions

### Decision 1: Embedded SQL Schema Migrations vs External Migration Tool
- **Choice:** Embedded SQL schema migrations inside Go (`internal/database/migrations/`).
- **Rationale:** Keeps deployment simple without requiring extra migration binaries or external tools inside Docker containers. Go reads embedded `.sql` scripts using `embed.FS` on startup and applies pending migrations safely inside DB transactions.
- **Alternatives Considered:** `golang-migrate/migrate` CLI (adds container build dependencies).

### Decision 2: 2-Level Subfolder Sharding
- **Choice:** `/storage/<uuid[0..1]>/<uuid[2..3]>/<uuid>`
- **Rationale:** Provides 65,536 subdirectories. Evenly distributes millions of files while keeping directory entry lookups extremely fast on EXT4/XFS filesystems.
- **Alternatives Considered:** Flat single folder (suffers severe filesystem performance degradation when storing > 10,000 files).

### Decision 3: Streaming Uploads with On-The-Fly Quota & SHA256 Hashing
- **Choice:** Use `io.TeeReader` and `crypto/sha256` to compute file hash while streaming body bytes to disk. Track written bytes against free user quota.
- **Rationale:** Prevents storing oversized files in memory or on disk before rejecting. Aborts connection with HTTP 413 Payload Too Large and deletes temp fragment immediately if quota is breached.
- **Alternatives Considered:** Buffer full file in RAM (risks OOM crashes under high load or large file uploads).

## Risks / Trade-offs

- [Disk Space Limits] → Pre-allocate and verify storage path permissions on app startup.
- [Unfinished Upload Temp Files] → Clean up partial temp files immediately on upload error or connection break; purge orphan temp files on startup.
- [Database Connection Bottlenecks] → Configure database connection pool with max connection limits and health checks.
