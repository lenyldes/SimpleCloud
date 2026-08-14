## 1. Continuous Integration & Docker Network Setup

- [x] 1.1 Create GitHub Actions CI workflow (`.github/workflows/ci.yml`) with `golangci-lint` and Go test suite against PostgreSQL 16 service container.
- [x] 1.2 Update `docker-compose.yml` to connect `web-frontend` to the external `caddy-public` network.

## 2. Continuous Deployment & Documentation

- [x] 2.1 Create GitHub Actions CD deploy workflow (`.github/workflows/deploy.yml`) for automated SSH deployment to `pi5server`.
- [x] 2.2 Create Caddy setup documentation (`docs/CADDY_GUIDE.md`) detailing `test-cloud.lenyldes.ru` routing configuration and Caddy reload instructions.
