## MODIFIED Requirements

### Requirement: Cross-Site Request Forgery Origin Validation
The system SHALL verify the `Origin` header on all state-changing (mutating) endpoints — login, logout, file upload, file deletion, folder creation, folder deletion. When an `Origin` header is present, its host MUST match the request host (taking `X-Forwarded-Host` into account); mismatches SHALL be rejected with `403 Forbidden`. The reverse proxy configuration SHALL preserve the client's original port using `$http_host` for `Host` and `X-Forwarded-Host`, and the origin validation middleware SHALL reliably validate origin against forwarded host preserving custom ports.

#### Scenario: Mutating request with mismatched Origin
- **WHEN** an authenticated mutating API request carries an `Origin` header whose host differs from `X-Forwarded-Host` (or `Host` when the forwarded header is absent)
- **THEN** the system SHALL respond with `403 Forbidden` and MUST NOT perform the mutation.

#### Scenario: Mutating request with same-origin Origin
- **WHEN** a mutating API request carries an `Origin` header whose host matches the request host
- **THEN** the system SHALL proceed with normal authentication and handling.

#### Scenario: Request without Origin header
- **WHEN** a mutating API request carries no `Origin` header (e.g. non-browser clients, same-page requests that omit it)
- **THEN** the CSRF check SHALL pass and the request SHALL be processed normally, with `SameSite=Lax` remaining as the second defense line.

#### Scenario: Mutating request with custom port in Origin and X-Forwarded-Host
- **WHEN** a client on a custom port sends a mutating request with `Origin: http://localhost:32214` and reverse proxy sends `X-Forwarded-Host: localhost:32214`
- **THEN** the system SHALL recognize matching hosts and proceed with normal processing rather than rejecting with CSRF error.
