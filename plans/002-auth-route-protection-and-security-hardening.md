# Plan 002: Enforce JWT authentication across all business REST routes and harden auth security

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in `api/cmd/server/server.go`, `api/cmd/bootstrap/bootstrap.go`, and `api/internal/modules/auth/...` against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security | bug
- **Covers findings**: `SEC-01`, `SEC-03`, `SEC-05`, `BUG-08`
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

All 44 `/api/v1/*` business endpoints across `portfolio`, `lease`, `payment`, `document`, `inspection`, and `accounting` are mounted directly onto the root `http.ServeMux` in `api/cmd/server/server.go:114-119` without `authhttp.RequireAuth`, leaving every business endpoint publicly accessible without a Bearer token. In addition, `RejectUser` revokes refresh tokens but never revokes `mcp_api_tokens`, `RequireAuthFunc` does not verify whether a user with a 15-minute JWT was rejected after issuance, `PasswordHasher` has no password length bounds or Argon2id concurrency cap (allowing 32 concurrent logins to allocate 2 GiB of RAM) and skips Argon2id on unknown emails (creating a timing oracle), and `BootstrapSuperAdmin`/`Signup` ignore non-`ErrUserNotFound` DB errors while `bootstrap.go` trims password whitespace.

## Current state

- `api/cmd/server/server.go:109-125` — mounts all 6 domain modules on `mux` without authentication middleware:
  ```go
  	mux := http.NewServeMux()
  	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
  		httpx.OK(w, map[string]string{"status": "ok"})
  	})
  	authMod.RegisterHTTPRoutes(mux)
  	portfolioMod.RegisterHTTPRoutes(mux)
  	leaseMod.RegisterHTTPRoutes(mux)
  	paymentMod.RegisterHTTPRoutes(mux)
  	documentMod.RegisterHTTPRoutes(mux)
  	inspectionMod.RegisterHTTPRoutes(mux)
  	accountingMod.RegisterHTTPRoutes(mux)
  ```
- `api/internal/modules/auth/interfaces/http/middleware.go:26-50` — defines `RequireAuth` and `RequireAuthFunc`, which validate JWT signatures statelessly:
  ```go
  func RequireAuth(jwtSvc *infrastructure.JwtService) func(http.Handler) http.Handler {
  	return func(next http.Handler) http.Handler {
  		return RequireAuthFunc(jwtSvc, next.ServeHTTP)
  	}
  }
  ```
- `api/internal/modules/auth/domain/mcp_token.go:26-29` — `MCPTokenRepository` lacks `RevokeAllUserMCPTokens`:
  ```go
  type MCPTokenRepository interface {
  	SaveMCPToken(ctx context.Context, token MCPApiToken) error
  	FindMCPTokenByHash(ctx context.Context, tokenHash string) (*MCPApiToken, error)
  }
  ```
- `api/internal/modules/auth/application/service.go:224-248` — `RejectUser` revokes refresh tokens inside `s.uow.WithinTransaction` but does not revoke MCP tokens:
  ```go
  	if err := s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
  		if err := repos.Users.UpdateUser(txCtx, *target); err != nil {
  			return err
  		}
  		return repos.RefreshTokens.RevokeAllUserRefreshTokens(txCtx, target.ID, now)
  	}); err != nil {
  ```
- `api/internal/modules/auth/application/service.go:122-133, 168-170, 275-284` — `BootstrapSuperAdmin` and `Signup` ignore `err != nil && !errors.Is(err, domain.ErrUserNotFound)` from `FindUserByEmail`, and `Login` returns immediately on `FindUserByEmail` error without dummy Argon2id verification.
- `api/internal/modules/auth/infrastructure/password_hasher.go:18-50` — `PasswordHasher` runs `argon2.IDKey` with 64 MiB memory per call with no max password length or concurrency limit.
- `api/cmd/bootstrap/bootstrap.go:114-116` — trims whitespace from `--password` (`password := strings.TrimSpace(cmd.String("password"))`).

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Auth tests | `cd api && go test -v ./internal/modules/auth/...` | exit 0, all pass |
| Full tests | `cd api && go test ./...` | exit 0, all pass |

## Scope

**In scope**:

- `api/cmd/server/server.go`
- `api/cmd/bootstrap/bootstrap.go`
- `api/cmd/main_test.go`
- `api/internal/modules/auth/domain/mcp_token.go`
- `api/internal/modules/auth/infrastructure/memory/mcp_token_repo.go`
- `api/internal/modules/auth/infrastructure/postgres/mcp_token_repo.go`
- `api/internal/modules/auth/infrastructure/password_hasher.go`
- `api/internal/modules/auth/infrastructure/security_test.go`
- `api/internal/modules/auth/application/service.go`
- `api/internal/modules/auth/application/service_test.go`
- `api/internal/modules/auth/interfaces/http/middleware.go`
- `api/internal/modules/auth/module.go`

**Out of scope**:

- Non-auth module HTTP handlers (`portfolio`, `lease`, `payment`, `document`, `inspection`, `accounting`) — protect them centrally in `api/cmd/server/server.go` so existing module-level unit tests continue to work unchanged.

## Git workflow

- Branch: `advisor/002-auth-route-protection-and-security-hardening`
- Commit message style: `fix(auth): enforce JWT middleware on business routes and harden auth security`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Add `RevokeAllUserMCPTokens` to `MCPTokenRepository` and call it in `RejectUser` (`SEC-03`)

1. In `api/internal/modules/auth/domain/mcp_token.go`, add `RevokeAllUserMCPTokens(ctx context.Context, userID string, revokedAt time.Time) error` to `MCPTokenRepository`.
2. Implement `RevokeAllUserMCPTokens` in:
   - `api/internal/modules/auth/infrastructure/memory/mcp_token_repo.go`: iterate `r.mcpByHash` under write lock and set `tok.RevokedAt = &revokedAt` when `tok.UserID == userID && tok.RevokedAt == nil`.
   - `api/internal/modules/auth/infrastructure/postgres/mcp_token_repo.go`: execute `UPDATE mcp_api_tokens SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL` via `platformpg.Conn(ctx, r.db).Exec(ctx, ...)`.
3. In `api/internal/modules/auth/application/service.go` inside `RejectUser`'s `s.uow.WithinTransaction` callback (lines 238–243), call `repos.MCPTokens.RevokeAllUserMCPTokens(txCtx, target.ID, now)` after `RevokeAllUserRefreshTokens`.

**Verify**: `cd api && go test ./internal/modules/auth/...` → exit 0.

### Step 2: Harden `PasswordHasher` and `auth/application/service.go` (`SEC-05`, `BUG-08`)

1. In `api/internal/modules/auth/infrastructure/password_hasher.go`:
   - Add a constant `MaxPasswordBytes = 128` and a buffered semaphore `sem chan struct{}` (capacity `8`, initialized in `NewArgon2Hasher()`) to bound concurrent 64 MiB Argon2id evaluations.
   - Reject passwords exceeding `MaxPasswordBytes` in `HashPassword` and `VerifyPassword`.
   - Acquire `h.sem <- struct{}{}` / `defer func() { <-h.sem }()` around `argon2.IDKey` in both `HashPassword` and `VerifyPassword`.
2. In `api/internal/modules/auth/application/service.go`:
   - Precompute a dummy Argon2id hash once in `NewService` (`dummyHash, _ := hasher.HashPassword(secret.New("dummy-constant-time-password"))`).
   - In `Login` (lines 275–278), when `s.userRepo.FindUserByEmail` returns an error, still call `_, _ = s.hasher.VerifyPassword(cmd.Password, s.dummyHash)` before returning `domain.ErrInvalidCredentials`.
   - In `Signup` and `BootstrapSuperAdmin`, validate password length (`len(password) >= 8 && len(password) <= 128`, returning `domain.ErrValidation` otherwise) and check `if err != nil && !errors.Is(err, domain.ErrUserNotFound) { return UserDTO{}, err }` after `FindUserByEmail`.
3. In `api/cmd/bootstrap/bootstrap.go:115`, replace `password := strings.TrimSpace(cmd.String("password"))` with `password := cmd.String("password")` (only check `strings.TrimSpace(password) == ""` for empty validation).

**Verify**: `cd api && go test -v ./internal/modules/auth/... ./cmd/...` → exit 0.

### Step 3: Verify active user approval in `RequireAuth` and protect all `/api/v1/*` business routes (`SEC-01`, `SEC-03`)

1. In `api/internal/modules/auth/application/service.go`, add a method `RequireApprovedUser(ctx context.Context, userID string) (*domain.User, error)` that loads the user by ID and returns `domain.ErrAccountNotApproved` if `!user.IsApproved()`.
2. In `api/internal/modules/auth/interfaces/http/middleware.go`, add an optional user-status validator function or `RequireApprovedAuth(jwtSvc *infrastructure.JwtService, checkApproved func(ctx context.Context, userID string) error) func(http.Handler) http.Handler` that returns `401 Unauthorized` on missing/invalid JWT and `403 Forbidden` if `checkApproved != nil` returns an error. Expose `RequireAuthMiddleware() func(http.Handler) http.Handler` on `auth.Module` (`api/internal/modules/auth/module.go`).
3. In `api/cmd/server/server.go:109-125`, register the 6 business modules (`portfolioMod`, `leaseMod`, `paymentMod`, `documentMod`, `inspectionMod`, `accountingMod`) on a `protectedMux := http.NewServeMux()`, and mount it onto the root `mux` wrapped with `authMod.RequireAuthMiddleware()` (while keeping `/healthz`, `/api/v1/auth/*`, `/api/v1/admin/*`, `/mcp`, and `/.well-known/oauth-protected-resource` on `mux`).

**Verify**: `cd api && go vet ./... && go test ./...` → exit 0.

## Test plan

- In `api/internal/modules/auth/application/service_test.go`, add tests verifying:
  - `RejectUser` revokes all active MCP tokens for the rejected user (`ValidateMCPBearer` fails immediately).
  - `Signup` rejects passwords shorter than 8 bytes or longer than 128 bytes with `domain.ErrValidation`.
- In `api/cmd/main_test.go`, add a test constructing `buildHTTPHandler` (or testing the protected mux) and asserting that `GET /api/v1/entities` without an `Authorization` header returns `401 Unauthorized` while `GET /healthz` returns `200 OK`.
- Verification: `cd api && go test -v ./internal/modules/auth/... ./cmd/...` → all pass.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test ./...` exits 0
- [ ] Unauthenticated requests to `/api/v1/entities` on the server mux return `401 Unauthorized`
- [ ] `RejectUser` revokes the target user's MCP tokens in both memory and Postgres repositories
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If any existing test in `api/internal/modules/auth/application/service_test.go` uses a password shorter than 8 characters (check `service_test.go` first; if it uses `"Secret123!"` or similar >= 8 chars, 8 bytes is safe; if shorter, adjust existing test fixtures in `service_test.go` only).
- If any file in "Current state" has drifted from the excerpts above.

## Maintenance notes

- Any new bounded-context module added to `api/cmd/server/server.go` must register its routes on `protectedMux` rather than the unauthenticated root `mux`.
