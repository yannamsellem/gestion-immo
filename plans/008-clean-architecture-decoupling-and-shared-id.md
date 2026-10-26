# Plan 008: Consolidate duplicated ID generation and decouple cross-module domain/infrastructure imports

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in `payment/domain/allocation_service.go`, `auth/application/service.go`, and the 7 module `generateID` helpers against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: LOW
- **Depends on**: `plans/002-auth-route-protection-and-security-hardening.md`, `plans/005-portfolio-lease-and-payment-correctness.md`, `plans/006-accounting-live-portfolio-and-balance-sync.md`
- **Category**: tech-debt
- **Covers findings**: `ARCH-01`
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

Every one of the 7 modules (`auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, `accounting`) copy-pastes its own `generateID(prefix string) (string, error)` helper at the bottom of `application/service.go`. In addition, `auth/application/service.go` directly imports `auth/infrastructure` concrete structs (`*infrastructure.JwtService`, `*infrastructure.PasswordHasher`) instead of depending on domain/application interfaces, violating the Clean Architecture dependency rule mandated in Section 3.1 of the design spec.

## Current state

- `generateID` duplicated across all 7 modules:
  - `api/internal/modules/auth/application/service.go:537-543`
  - `api/internal/modules/portfolio/application/service.go:518-524`
  - `api/internal/modules/lease/application/service.go:500-506`
  - `api/internal/modules/payment/application/service.go:674-680`
  - `api/internal/modules/document/application/service.go:880-886`
  - `api/internal/modules/inspection/application/service.go:833-839`
  - `api/internal/modules/accounting/application/service.go:788-794`
- `api/internal/modules/auth/application/service.go:13, 81-107`:
  ```go
  	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
  ...
  type Service struct {
  	userRepo    domain.UserRepository
  	refreshRepo domain.RefreshTokenRepository
  	mcpRepo     domain.MCPTokenRepository
  	uow         domain.UnitOfWork
  	jwtSvc      *infrastructure.JwtService
  	hasher      *infrastructure.PasswordHasher
  }
  ```

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Test | `cd api && go test -race ./...` | exit 0 |

## Scope

**In scope**:

- `api/internal/platform/id/id.go` (create)
- `api/internal/platform/id/id_test.go` (create)
- `api/internal/modules/auth/domain/user.go` (or `api/internal/modules/auth/domain/security.go` create)
- `api/internal/modules/auth/application/service.go`
- `api/internal/modules/portfolio/application/service.go`
- `api/internal/modules/lease/application/service.go`
- `api/internal/modules/payment/application/service.go`
- `api/internal/modules/document/application/service.go`
- `api/internal/modules/inspection/application/service.go`
- `api/internal/modules/accounting/application/service.go`

**Out of scope**:

- Changing any public HTTP or MCP contract.

## Git workflow

- Branch: `advisor/008-clean-architecture-decoupling-and-shared-id`
- Commit message style: `refactor(api): extract platform/id helper and decouple auth application from infrastructure`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Extract `api/internal/platform/id/id.go` and replace the 7 duplicated `generateID` functions (`ARCH-01`)

1. Create `api/internal/platform/id/id.go` with `Generate(prefix string) (string, error)` using 16 cryptographically random bytes (`crypto/rand` + `encoding/hex`, returning `fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf[:]))`), plus `api/internal/platform/id/id_test.go`.
2. In all 7 `application/service.go` files (`auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, `accounting`), replace calls to `generateID("...")` with `id.Generate("...")` and delete the private `generateID` function (and unused `crypto/rand` / `encoding/hex` imports).

**Verify**: `cd api && go vet ./... && go test ./...` → exit 0.

### Step 2: Decouple `auth/application` from `auth/infrastructure` via domain interfaces (`ARCH-01`)

1. Create `api/internal/modules/auth/domain/security.go` defining:
   ```go
   package domain

   import "github.com/yannamsellem/gestion-immo/internal/platform/secret"

   type AccessTokenManager interface {
   	GenerateAccessToken(user User) (secret.String, error)
   	ValidateAccessToken(rawToken string) (*Claims, error)
   }

   type SecretHasher interface {
   	HashPassword(plain secret.String) (string, error)
   	VerifyPassword(plain secret.String, encodedHash string) (bool, error)
   	HashToken(rawToken secret.String) string
   }
   ```
2. In `api/internal/modules/auth/application/service.go`, replace `*infrastructure.JwtService` and `*infrastructure.PasswordHasher` with `domain.AccessTokenManager` and `domain.SecretHasher`, removing the `"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"` import from `auth/application/service.go`.

**Verify**: `cd api && go vet ./... && go test -v ./internal/modules/auth/...` → exit 0.

## Test plan

- `api/internal/platform/id/id_test.go` verifies prefix formatting and uniqueness across 1,000 generated IDs.
- All existing tests in `auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, and `accounting` pass unchanged.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test -race ./...` exits 0
- [ ] `grep -rn "func generateID(" api/internal/modules/` returns 0 matches
- [ ] `grep -rn "auth/infrastructure" api/internal/modules/auth/application/service.go` returns 0 matches
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If any file in "Current state" has drifted from the excerpts above.
- If a step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- New modules should use `id.Generate("<prefix>")` from `internal/platform/id` rather than defining module-local random hex generators.
