## ADDED Requirements

### Requirement: No Host-Published Backend Ports
The Docker Compose deployment MUST NOT publish the `storage-service` API port or the PostgreSQL port on the host. All external client traffic SHALL enter exclusively through the `web-frontend` reverse proxy (host port `32214` / Caddy), so nginx rate limiting and request size limits cannot be bypassed and the database is not reachable outside the Docker network.

#### Scenario: Direct backend port access refused
- **WHEN** a client outside the Docker host network attempts to connect to the former backend ports (`8080` storage API, `5432` PostgreSQL) on the server
- **THEN** the connection SHALL be refused because no host port mapping exists

#### Scenario: Application remains reachable through the frontend proxy
- **WHEN** `docker compose up -d` is executed after the port mappings are removed
- **THEN** the application SHALL continue to serve login, upload, download, and browsing flows through `web-frontend` on host port `32214` and via the Caddy-routed domain, with `storage-service` reachable from other containers only by its Docker network service name
