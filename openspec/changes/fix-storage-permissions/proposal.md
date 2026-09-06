## Why

During user acceptance testing (BUGS.md, finding BUG-3), attempting to upload files via Drag-and-Drop or the "Upload" button fails with the following runtime error:
`{"error": "failed to save file: failed to create directory structure /storage/7c/f0: mkdir /storage/7c: permission denied"}`

This defect occurs because `docker-compose.yml` mounts the host directory `./data/storage` into `/storage`. When Docker creates this directory or mounts it from the host, the directory is owned by `root:root` (or the host user UID 1000) with `0755` permissions. Meanwhile, the container process runs as non-root user `app` created in Alpine via `adduser -S app` with system UID 100. As a result, the container process has only "other" (read/execute) permissions on `/storage` and cannot create subdirectories or write files. Furthermore, the user explicitly requires physical, unhindered access to uploaded files on the host in `./data/storage`.

## What Changes

- **Add Docker Container Entrypoint with `su-exec`**:
  - Update `services/storage-service/Dockerfile` to install Alpine's `su-exec` utility and configure the default `app` user with UID/GID 1000:1000.
  - Introduce `services/storage-service/entrypoint.sh` executed as root on container boot to read configurable `PUID` and `PGID` (defaulting to 1000:1000), ensure `/storage` ownership and `0775` permissions, and drop privileges to `PUID:PGID` via `exec su-exec "$PUID:$PGID" "$@"` before running `storage-service`.
- **Pass PUID/PGID in Docker Compose**:
  - Update `docker-compose.yml` to pass `PUID=${PUID:-1000}` and `PGID=${PGID:-1000}` into `storage-service` environment.
- **Fail-Fast Storage Directory Verification in Go**:
  - Add `EnsureStorageDir()` method to `storage.DiskEngine` that creates the base storage directory with `0775` and performs a probe write-and-remove check.
  - Invoke `EnsureStorageDir()` in `cmd/main.go` during server initialization so permission faults are caught immediately at boot rather than on the first upload request.
- **Set File and Directory Creation Permissions in Go**:
  - Update `services/storage-service/internal/storage/sharding.go` to create shard directories with `0775` and set saved files to mode `0664` (`os.Chmod`) so the host user can directly inspect and read uploaded files without `sudo`.
- **Pre-create Host Storage Directory in CI/CD Deploy**:
  - Update the SSH deploy script in `.github/workflows/ci.yml` to run `mkdir -p data/storage` on the VPS host prior to `docker compose up -d --build`.

## Capabilities

### New Capabilities
*(None)*

### Modified Capabilities
- `file-storage`: Add requirements for storage directory startup verification (`EnsureStorageDir`), directory creation mode `0775`, and file mode `0664` for host accessibility.
- `deployment-ci-cd`: Add requirements for `PUID`/`PGID` container entrypoint permission initialization via `su-exec` in `storage-service`, and host directory pre-creation in the deployment pipeline.

## Impact

- `services/storage-service/Dockerfile`: Adds `su-exec` package, sets up `entrypoint.sh`, manages `PUID`/`PGID`.
- `services/storage-service/entrypoint.sh`: New entrypoint script handling directory ownership and privilege drop.
- `docker-compose.yml`: Adds `PUID` and `PGID` environment variables for `storage-service`.
- `.github/workflows/ci.yml`: Adds `mkdir -p data/storage` to deployment step.
- `services/storage-service/internal/storage/sharding.go`: Adds `EnsureStorageDir()`, sets `0775` on dirs and `0664` on files.
- `services/storage-service/internal/storage/sharding_test.go`: Tests for `EnsureStorageDir()` and file permissions.
- `services/storage-service/cmd/main.go`: Calls `EnsureStorageDir()` at startup.
- `BUGS.md`: Resolves BUG-3.
