## Context

During file uploads (both Drag-and-Drop and the "Upload" button), the backend returns `mkdir /storage/7c: permission denied` (BUG-3). 
This stems from a permissions mismatch between:
1. The host filesystem where `./data/storage` is mounted into `/storage` in `docker-compose.yml`.
2. The container runtime where `storage-service` runs as Alpine system user `app` (UID 100).
3. The host user (e.g. `pi` or `ubuntu` on `pi5server`, standard UID 1000) who needs uninhibited physical access to the files on the server directly in the project directory.

See `proposal.md` for motivation and background.

## Goals / Non-Goals

**Goals:**
- Eliminate `permission denied` errors when creating sharded directories and saving uploaded files.
- Guarantee that all files and subdirectories created in `./data/storage` are directly accessible, readable, and manageable by the host user (UID 1000) without requiring `sudo`.
- Keep the server process running as a non-root user for container security hygiene.
- Provide a fail-fast startup probe (`EnsureStorageDir`) that detects storage permission defects at container boot rather than at the first user upload.
- Ensure automated VPS deployment (`ci.yml`) creates `data/storage` under host user ownership before Docker Compose boots.

**Non-Goals:**
- Switching to Docker named volumes (`volumes: storage_data:/storage`): Explicitly prohibited because it isolates files inside `/var/lib/docker/volumes/...`, breaking direct access in the project directory.
- Running the production Go server process as `root`: Excluded for security reasons; root privileges are used strictly in the entrypoint to fix volume permissions before dropping to `PUID:PGID`.
- Modifying frontend upload logic: Frontend event handling is already verified; this change addresses backend filesystem, container permissions, and deployment orchestration.

## Decisions

### 1. Alpine `su-exec` in Container Entrypoint
- **Decision**: Install `su-exec` in `alpine:3.21`, introduce `entrypoint.sh` running as initial root to check/chown `/storage` to `PUID:PGID` and `chmod 775`, then drop privileges via `exec su-exec "$PUID:$PGID" "$@"`.
- **Rationale**: If Docker daemon previously created `./data/storage` as `root:root 0755` on the server, a container running strictly as non-root cannot fix the volume permissions. The entrypoint pattern is the industry standard (used by official PostgreSQL, Nextcloud, and LinuxServer images) to ensure self-healing permissions while running the actual application as non-root.
- **Alternatives Considered**:
  - *Run container entirely as root*: Rejected. Insecure, and all files created on host would be owned by `root:root 0600`, preventing the host user from reading or modifying them without `sudo`.
  - *Hardcode UID 1000 in Dockerfile with `USER app` without entrypoint*: Rejected because if `./data/storage` on the host is owned by root, the container still gets `permission denied`.
  - *Require manual `chmod 777` on host*: Rejected as fragile and prone to regressions during clean deployments or rebuilds.

### 2. Configurable `PUID` and `PGID` with 1000:1000 Defaults
- **Decision**: Support `PUID` and `PGID` environment variables in `entrypoint.sh` and `docker-compose.yml`, defaulting to `1000:1000`.
- **Rationale**: On almost all Linux systems (Raspberry Pi OS, Debian, Ubuntu), the primary non-root user (`pi`, `ubuntu`, `admin`) has UID 1000 and GID 1000. Matching this identity inside the container ensures files written to the bind mount are naturally owned by the host user.

### 3. Fail-Fast Startup Verification (`EnsureStorageDir`)
- **Decision**: Add `EnsureStorageDir() error` to `storage.DiskEngine` and call it during service startup in `cmd/main.go`. The method ensures the directory exists (`os.MkdirAll(..., 0775)`) and tests write access with a probe file (`.write-probe-*`).
- **Rationale**: If permissions or storage paths are misconfigured, the container should exit immediately with an informative diagnostic message rather than passing `/health` checks and failing silently until a user upload occurs.

### 4. Explicit Permissions on Sharded Paths (`0775` dirs, `0664` files)
- **Decision**: In `internal/storage/sharding.go`, create parent directories with `os.MkdirAll(dir, 0775)` and apply `os.Chmod(targetPath, 0664)` to saved files.
- **Rationale**: Standard Go `os.CreateTemp` creates files with `0600` (read/write only by owner). Changing the mode to `0664` ensures that any user or group on the host can inspect, backup, and read the stored files without permission blocks.

### 5. Deployment Pre-creation (`mkdir -p data/storage`)
- **Decision**: In `.github/workflows/ci.yml`, insert `mkdir -p data/storage` into the SSH deployment script before running `docker compose up -d --build`.
- **Rationale**: Prevents the Docker daemon from creating the host directory as `root:root` when the repository is cloned or checked out on a fresh host.

## Risks / Trade-offs

- **[Risk] Initial root execution in container entrypoint** → **Mitigation**: The entrypoint performs only ownership and directory setup, then permanently drops privileges using `su-exec` before executing the server process.
- **[Risk] Host user on a different machine has a non-1000 UID** → **Mitigation**: `PUID` and `PGID` are parameterized in `docker-compose.yml` (`PUID=${PUID:-1000}`) and can be overridden via `.env`.
- **[Risk] Existing Go unit tests break if `/storage` is not writable on test machines** → **Mitigation**: All existing unit and integration tests already use `t.TempDir()` or `/tmp/simplecloud_test_storage` where full write permissions exist. `EnsureStorageDir` will be thoroughly unit-tested for both success and error branches.
