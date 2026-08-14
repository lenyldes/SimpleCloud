## Context

See `proposal.md`. SimpleCloud consists of three containers (`postgres`, `storage-service`, `web-frontend`). `web-frontend` runs Nginx on internal port 80. The host VPS (`pi5server`) already runs a central Caddy reverse proxy container connected to a shared Docker network named `caddy-public`.

## Goals / Non-Goals

**Goals:**
- Provide GitHub Actions workflow `.github/workflows/ci.yml` for linting and Go unit/integration testing with PostgreSQL service container.
- Provide GitHub Actions workflow `.github/workflows/deploy.yml` for automated SSH deployment to `pi5server`.
- Configure `docker-compose.yml` to attach `web-frontend` to external network `caddy-public`.
- Document Caddy routing configuration (`docs/CADDY_GUIDE.md`) and deployment steps.

**Non-Goals:**
- Deploying Caddy as a new container inside SimpleCloud's `docker-compose.yml` (using existing `caddy-proxy` on VPS).
- Dynamic DNS configuration or domain registration.

## Decisions

1. **Docker Network Sharing (`caddy-public`)**:
   - *Decision*: Attach `web-frontend` service to `caddy-public` network declared as `external: true` in `docker-compose.yml`.
   - *Rationale*: Allows host Caddy proxy to resolve container hostname `simplecloud-web-frontend:80` without binding host ports or exposing internal ports to host `localhost`.
   - *Alternative Considered*: Binding port `32214` to host localhost and proxying to `localhost:32214`. Rejected because using direct container networking is cleaner and matches existing VPS services (`rocketchat`, `nextcloud`).

2. **GitHub Actions SSH Deployment (`appleboy/ssh-action`)**:
   - *Decision*: Use `appleboy/ssh-action` in `.github/workflows/deploy.yml` with SSH secrets.
   - *Rationale*: Standard, robust mechanism to run deployment commands (`git pull`, `docker compose up -d --build`, Caddy reload) on remote VPS.

3. **Caddy Reload Command**:
   - *Decision*: Include automated Caddy configuration reload `docker compose exec -T caddy caddy reload` in deployment script.
   - *Rationale*: Ensures zero-downtime certificate issuance and routing update for `test-cloud.lenyldes.ru`.

## Risks / Trade-offs

- **[Risk]** `caddy-public` network does not exist on local dev environment when running `docker compose up`.
  - → *Mitigation*: Document that `caddy-public` can be created locally via `docker network create caddy-public` or made optional in local override compose files.
- **[Risk]** SSH secret misconfiguration or network timeout during deployment.
  - → *Mitigation*: Workflow step includes timeout limits and detailed step reporting.
