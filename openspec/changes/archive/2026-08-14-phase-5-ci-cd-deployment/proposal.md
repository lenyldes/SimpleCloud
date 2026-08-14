## Why

SimpleCloud currently lacks automated continuous integration testing and automated continuous deployment to the production server. Setting up automated linting and test execution on GitHub Actions along with continuous SSH deployment to the `pi5server` VPS ensures code reliability, enforces quality standards (85%+ code coverage), and eliminates manual deployment errors.

## What Changes

- **CI Pipeline (`.github/workflows/ci.yml`)**: Run `golangci-lint` and `go test -v -cover ./...` with a live PostgreSQL 16 service container in GitHub Actions.
- **CD Pipeline (`.github/workflows/deploy.yml`)**: Automated SSH deployment to `pi5server` upon push/merge to `main`.
- **Docker Compose Networking**: Expose `web-frontend` container to the external `caddy-public` Docker network on `pi5server`.
- **Caddy Reverse Proxy Configuration (`docs/caddy/Caddyfile.snippet`)**: Add routing directive `test-cloud.lenyldes.ru` -> `reverse_proxy simplecloud-web-frontend:80` for the host Caddy server.

## Capabilities

### New Capabilities
- `deployment-ci-cd`: Automated CI/CD pipelines in GitHub Actions and VPS deployment architecture via Caddy reverse proxy on `pi5server`.

### Modified Capabilities
*(None)*

## Impact

- `.github/workflows/ci.yml` and `.github/workflows/deploy.yml` created.
- `docker-compose.yml` updated to include `caddy-public` external network on `web-frontend`.
- Documentation added for Caddy reverse proxy setup (`docs/CADDY_GUIDE.md`).
