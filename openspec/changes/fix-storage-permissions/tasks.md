## 1. Storage & Permissions Unit Tests (RED Phase - Test Agent)

- [x] 1.1 Add unit tests for `EnsureStorageDir` in `services/storage-service/internal/storage/sharding_test.go` covering successful directory creation with probe file verification and failure on unwritable paths. Verify tests fail to compile or run (RED state).
- [x] 1.2 Add unit tests for sharded directory permissions (`0775`) and saved file permissions (`0664`) in `services/storage-service/internal/storage/sharding_test.go`. Verify tests fail initially (RED state).
- [x] 1.3 Add static regression tests in `services/storage-service/cmd/main_test.go` verifying that `Dockerfile` installs `su-exec`, `entrypoint.sh` exists and is executable, and `docker-compose.yml` passes `PUID` and `PGID`. Verify tests fail initially (RED state).

## 2. Storage & Permissions Implementation (GREEN Phase - Code Agent)

- [x] 2.1 Implement `EnsureStorageDir()` in `services/storage-service/internal/storage/sharding.go` and invoke it in `services/storage-service/cmd/main.go` on server startup. Verify Task 1.1 unit tests pass (GREEN state).
- [x] 2.2 Update `Save()` in `services/storage-service/internal/storage/sharding.go` to create parent directories with mode `0775` and set saved files to mode `0664` (`os.Chmod`). Verify Task 1.2 unit tests pass (GREEN state).
- [x] 2.3 Create `services/storage-service/entrypoint.sh` with `su-exec` and `PUID`/`PGID` privilege dropping, update `services/storage-service/Dockerfile` to install `su-exec` and wire `entrypoint.sh`, and update `docker-compose.yml` to pass `PUID` and `PGID`. Verify Task 1.3 tests pass (GREEN state).
- [x] 2.4 Update the SSH deployment step in `.github/workflows/ci.yml` to run `mkdir -p data/storage` prior to `docker compose up -d --build`.

## 3. Verification & Compliance (Audit Agent)

- [ ] 3.1 Run `go test -v -cover ./...` across `services/storage-service` to verify 100% pass rate and maintain 85%+ coverage in `internal/*`.
- [ ] 3.2 Verify code formatting with `gofmt` and run `golangci-lint` to ensure build hygiene.
