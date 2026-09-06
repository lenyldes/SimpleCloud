## MODIFIED Requirements

### Requirement: Automated Continuous Deployment
The system SHALL execute automated SSH deployment to `pi5server` VPS upon merging code into `main` after CI checks pass. The deployment script SHALL ensure host storage directory `./data/storage` is pre-created by the deployment user before rebuilding and launching Docker containers.

#### Scenario: Successful CD deployment to VPS
- **WHEN** code is merged into `main`
- **THEN** GitHub Actions SSHs into `pi5server`, pulls the latest code, ensures `data/storage` exists with host ownership, and runs `docker compose up -d --build` to update services.

## ADDED Requirements

### Requirement: Container Runtime Permissions and Entrypoint Privilege Drop
The `storage-service` container SHALL support configurable non-root user IDs via `PUID` and `PGID` environment variables (defaulting to 1000:1000). On container startup, an entrypoint script SHALL ensure the mounted storage directory `/storage` exists, is owned by `PUID:PGID`, and has permissions `0775`. The entrypoint SHALL drop root privileges using `su-exec` and execute the server binary as the non-root user `PUID:PGID`.

#### Scenario: Container boot initializes permissions and drops privileges
- **WHEN** the `storage-service` container starts up
- **THEN** `entrypoint.sh` sets `/storage` ownership to `PUID:PGID` (default 1000:1000) with mode `0775`, drops privileges via `su-exec`, and runs `storage-service` as `PUID:PGID`.

#### Scenario: Custom PUID and PGID support
- **WHEN** `PUID` and `PGID` environment variables are supplied to the container
- **THEN** the entrypoint configures directory ownership and drops privileges matching the specified numeric UID and GID.
