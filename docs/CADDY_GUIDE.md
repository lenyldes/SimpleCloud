# SimpleCloud - Caddy Reverse Proxy & Domain Setup Guide

This guide details how to integrate **SimpleCloud** with the central Caddy reverse proxy on the `pi5server` VPS to serve the application securely over HTTPS at `https://test-cloud.lenyldes.ru`.

---

## Architecture Overview

SimpleCloud's `web-frontend` container connects to the shared Docker network `caddy-public`. The main Caddy reverse proxy container running on `pi5server` is also attached to `caddy-public`. This allows Caddy to proxy incoming requests directly to `simplecloud-web-frontend:80` without binding host ports or exposing unencrypted ports publicly.

```
                  [ HTTPS Request: test-cloud.lenyldes.ru ]
                                     │
                                     ▼
                     ┌───────────────────────────────┐
                     │     Caddy Proxy Container     │
                     │    (Network: caddy-public)    │
                     └───────────────┬───────────────┘
                                     │
                                     ▼ (Reverse Proxy)
                     ┌───────────────────────────────┐
                     │   simplecloud-web-frontend    │
                     │          (Port 80)            │
                     └───────────────┬───────────────┘
                                     │
                                     ▼ (Internal Proxy /api/)
                     ┌───────────────────────────────┐
                     │   simplecloud-storage-service │
                     │         (Port 8080)           │
                     └───────────────────────────────┘
```

---

## 1. Prerequisites

1. Ensure the shared Docker network `caddy-public` exists on host VPS:
   ```bash
   docker network create caddy-public || true
   ```
2. Verify DNS record points `test-cloud.lenyldes.ru` to `pi5server` IP address.

---

## 2. Host Caddyfile Configuration

Add the following block to your central `Caddyfile` (or include it via snippet):

```caddy
test-cloud.lenyldes.ru {
    reverse_proxy simplecloud-web-frontend:80
}
```

---

## 3. Reloading Caddy Configuration

After adding the route snippet to the central Caddyfile, reload Caddy without downtime:

```bash
docker compose exec -T caddy caddy reload
# Or using standard Docker CLI:
docker exec caddy caddy reload
```

---

## 4. Environment Secrets Setup (GitHub Actions CD)

For automated deployment via `.github/workflows/deploy.yml`, configure the following repository secrets in GitHub (`Settings -> Secrets and variables -> Actions`):

- `VPS_HOST`: IP address or domain of `pi5server`.
- `VPS_USERNAME`: SSH username (e.g., `root` or `deploy`).
- `VPS_SSH_KEY`: Private SSH key authorized on `pi5server`.
- `VPS_SSH_PASSPHRASE`: (Optional) Passphrase for SSH key if encrypted.
- `VPS_PORT`: (Optional) Custom SSH port (defaults to `22`).

---

## 5. Troubleshooting & Verification

- **Verify Container Connectivity**:
  ```bash
  docker exec -it caddy ping simplecloud-web-frontend
  ```
- **Check Caddy Logs**:
  ```bash
  docker logs caddy --tail 50 -f
  ```
- **Local Dev without `caddy-public`**:
  If running locally without a global Caddy proxy, create the network before running Compose:
  ```bash
  docker network create caddy-public
  docker compose up -d
  ```
  Access locally at `http://localhost:32214`.
