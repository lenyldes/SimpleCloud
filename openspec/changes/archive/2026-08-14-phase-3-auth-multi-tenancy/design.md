## Context

See `proposal.md` for motivation and background.

SimpleCloud's `storage-service` is moving from Phase 2 mock user (`00000000-0000-0000-0000-000000000001`) to Phase 3 authentication and multi-tenancy. This design outlines the isolated architecture of the `internal/auth` package and context-based multi-tenancy enforcement across HTTP handlers and PostgreSQL queries.

## Goals / Non-Goals

**Goals:**
- Create an isolated `services/storage-service/internal/auth` package with 0 tight coupling to file handlers.
- Add database migration `000002_auth_schema.sql` updating `users` and introducing `user_sessions`.
- Implement HTTP-only Cookie session authentication (`simplecloud_session`) for browser clients.
- Provide HTTP middleware supporting both Cookie and `Authorization: Bearer` session extraction, injecting `user_id` into `r.Context()`.
- Enforce strict `WHERE user_id = $1` scoping in all SQL database operations.
- Implement startup admin account seeding using environment variables `ADMIN_EMAIL` and `ADMIN_PASSWORD` hashed with `bcrypt`.

**Non-Goals:**
- Stateful OAuth2 / OIDC social login providers (Google, GitHub) in Phase 3.
- Full stateless JWT RSA signing infrastructure in Phase 3 (deferred to Phase 6 API key / sync client expansion).

## Decisions

### 1. Isolated Module Architecture (`internal/auth`)
- **Decision:** Place all authentication logic, repository methods, handlers, and context helpers in `services/storage-service/internal/auth/`.
- **Rationale:** Prevents file handlers (`internal/handler/file.go`) from depending on authentication internals. File handlers only interact with `auth.GetUserIDFromContext(r.Context())`.
- **Alternatives Considered:** Embedding auth directly into `internal/handler/`. Rejected due to tight coupling and poor testability.

### 2. Session Authentication via HTTP-Only Cookies & Database Tokens
- **Decision:** Store session metadata in `user_sessions` table in PostgreSQL. Deliver session ID via `simplecloud_session` cookie (`HttpOnly: true`, `SameSite: http.SameSiteLaxMode`, `Path: "/"`).
- **Rationale:** HTTP-only cookies prevent XSS token theft in browser frontend (Phase 4). Storing sessions in PostgreSQL allows instant server-side session revocation on logout.
- **Alternatives Considered:** Pure JWT stored in `localStorage`. Rejected due to XSS vulnerability and inability to revoke compromised tokens instantly without a blacklist.

### 3. Password Hashing with Bcrypt
- **Decision:** Use `golang.org/x/crypto/bcrypt` with `bcrypt.DefaultCost` (cost 10) for hashing user passwords.
- **Rationale:** Standard, battle-tested password hashing mechanism in Go ecosystem.

### 4. Strongly Typed Context Keys
- **Decision:** Define private type `type contextKey string` in `internal/auth` with `const UserIDKey contextKey = "user_id"`. Provide package helper functions `auth.WithUserID(ctx, userID)` and `auth.GetUserIDFromContext(ctx)`.
- **Rationale:** Prevents context key collisions with third-party middleware or HTTP packages.

## Architectural Data Flow

```
                      ┌──────────────────────────────────────────┐
                      │            HTTP Request                  │
                      └────────────────────┬─────────────────────┘
                                           │
                                           ▼
                      ┌──────────────────────────────────────────┐
                      │    auth.RequireAuth Middleware (Go)      │
                      │                                          │
                      │ 1. Read Cookie simplecloud_session OR    │
                      │    Header Authorization: Bearer <token>  │
                      │ 2. Validate token in user_sessions DB    │
                      │ 3. Inject userID into r.Context()        │
                      └────────────────────┬─────────────────────┘
                                           │
                                           ▼
                      ┌──────────────────────────────────────────┐
                      │        File / Resource Handlers          │
                      │                                          │
                      │ userID, ok := auth.GetUserID(r.Context())│
                      │ db.GetFilesByUserID(ctx, userID)         │
                      └──────────────────────────────────────────┘
```

## Risks / Trade-offs

- **[Risk]** Database query overhead on every authenticated HTTP request for session lookup.
  - **Mitigation:** Index `user_sessions(token_hash)` and `user_sessions(expires_at)`. Session queries are fast indexed primary lookup (`< 1ms`).

- **[Risk]** Migration of existing single-tenant files from Phase 2 mock user ID.
  - **Mitigation:** Migration SQL updates existing mock files to point to the seeded admin user ID (`00000000-0000-0000-0000-000000000001`).

## Migration Plan

1. Run migration `000002_auth_schema.sql` adding `password_hash`, `role`, `is_active` to `users`, and creating `user_sessions`.
2. Seed default admin user on service startup if no users exist.
3. Update HTTP route registration in `cmd/main.go` to wrap `/api/v1/files/*` with `auth.RequireAuth(fileHandler...)`.
