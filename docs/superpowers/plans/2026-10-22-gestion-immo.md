# Gestion Immo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a full-stack French real estate and accounting management platform (Modular Go monolith + PostgreSQL + SvelteKit SSR + Cloudflare R2/Tunnel + native MCP server) that replaces spreadsheets, Drive folders, and EBP by unifying portfolio management, rent/cash tracking with negative position alerts, legal/GLI templating, photo-driven inspections via Antigravity MCP, and dynamic annual financial statements (*Comptes Annuels & Liasse Fiscale*).

**Architecture:** A Modular Monolith in Go structured by DDD Bounded Contexts (`internal/modules/{auth,portfolio,lease,payment,document,inspection,accounting}`) backed by a shared technical platform (`internal/platform/`), where each module strictly enforces Clean Architecture (`domain`, `application`, `infrastructure`, `interfaces/http`, `interfaces/mcp`) and exposes a constructor-injected `module.go`. The binary is driven by `github.com/urfave/cli/v3`, developer workflows are standardized through a root `Makefile`, and a SvelteKit SSR frontend (`shadcn-svelte`) on Cloudflare Pages communicates with the Go monolith and its embedded Streamable HTTP MCP server over a Cloudflare Tunnel (`cloudflared`).

**Tech Stack:** Go (latest stable, `github.com/urfave/cli/v3`, `stdlib net/http`, `github.com/jackc/pgx/v5`, `github.com/golang-migrate/migrate/v4`, `golang.org/x/crypto/argon2`), GNU Make (`Makefile`), PostgreSQL 16, SvelteKit (SSR, TypeScript, Tailwind CSS, `shadcn-svelte`), Cloudflare R2 (S3-compatible API), Cloudflare Tunnel (`cloudflared`), Gotenberg (HTML-to-PDF sidecar), Docker Compose, Terraform.

**Spec:** `docs/superpowers/specs/2026-10-21-gestion-immo-design.md`

## Global Constraints

- Latest stable Go version required.
- Code must be split into self-contained DDD business modules under `internal/modules/<module>` with shared non-business primitives in `internal/platform/`.
- Follow Clean Architecture, DDD, and Clean Code standards ALWAYS (`domain` layer inside every module has zero external or infrastructure dependencies).
- Constructor injection only (`NewX(...)` functions and per-module `NewModule(...)`; no global state or reflection DI containers).
- Use `github.com/urfave/cli/v3` as the CLI / binary manager (`serve`, `migrate up`, `migrate down`, `token create`).
- A comprehensive root `Makefile` is mandatory for Developer Experience (DX).
- Use immutable DTOs where practical (value structs without mutator methods; money represented as `AmountCents int64`, never `float64`).
- Never store plaintext passwords (hash with Argon2id).
- Never store plaintext refresh tokens (store SHA-256 hex digest in PostgreSQL).
- User signup (`POST /api/v1/auth/signup`) must create users in `PENDING_APPROVAL` status; login, token refresh, and MCP access are strictly forbidden (`403 Forbidden`) until explicitly approved by the `SUPER_ADMIN`.
- Keep JWT generation and validation isolated in `JwtService` (`internal/modules/auth/infrastructure/jwt_service.go`).
- Use proper semantic HTTP status codes (`200`, `201`, `204`, `400`, `401`, `403`, `404`, `409`, `422`, `500`).
- Never log passwords, JWTs, or refresh tokens (enforce `slog.LogValuer` redaction on sensitive types).
- Configuration must come exclusively from environment variables; no hard-coded secrets.
- Database migrations must use `golang-migrate`.
- HTTP router must use Go `stdlib net/http` (`http.NewServeMux` with Go 1.22+ method/path patterns).
- PostgreSQL driver must use `pgx` (`github.com/jackc/pgx/v5/pgxpool`).
- Frontend design system must use `shadcn-svelte`.
- Zero paid LLM API calls in the application; all AI capabilities go through the embedded MCP server consumed by Antigravity.
- Open Banking (DSP2) must remain 100% optional and disabled by default (`ENABLE_OPEN_BANKING=false`).

## Review Focus

1. **Overpayment or partial payment with explicit month in transfer label:** When a tenant owes January (`800_00`), February (`800_00`), and March (`800_00`) and transfers `1000_00` in March with label `"Virement Loyer Mars 2026"`, March (`800_00`) must be marked `PAYE` first and the remaining `200_00` must spill over via FIFO to the oldest unpaid month (January becomes `PARTIEL` with `600_00` remaining due). Covered in Task 5 (`TestAllocatePayment_ExplicitMonthWithOverpaymentSpillsToFIFO`).
2. **Disabling accounting lines flips net result sign and N-1 zero denominator:** Toggling off revenue or expense lines in `FinancialStatementCalculator` must dynamically recompute all parent subtotals, *Valeur Ajoutée*, *EBE*, *Résultat d'exploitation*, and *Résultat de l'exercice* across both Compte de Résultat and Bilan Passif, and any line with `NetNMinus1Cents == 0` must output variation `"N/S"` instead of dividing by zero. Covered in Task 8 (`TestCalculateStatement_DisabledLineFlipsResultAndZeroPrevYearReturnsNS`).
3. **Commercial lease VAT and share nominal values without floating-point drift:** Computing 20% VAT on commercial rents (`LOYERS COMMERCIAUX`) and share capital totals must use exact integer cents (`AmountCents int64`) so totals never drift by 1 centime. Covered in Task 4 (`TestCreateCommercialRentCall_VATExactCentsNoFloatDrift`).
4. **Secret leakage via structured logging or DB persistence:** Passing auth requests, config structs, or token responses to `slog` must redact passwords, JWTs, and refresh tokens as `"[REDACTED]"`, and raw refresh tokens must never match stored DB values. Covered in Task 2 (`TestAuthSecurity_NeverLogsOrStoresPlaintextSecrets`).
5. **Exit inspection on legacy lease without structured entry inspection:** Generating or comparing a move-out inspection (*état des lieux sortant*) for a migrated lease that has `EntryInspectionID == nil` must succeed in standalone mode without panicking or failing validation. Covered in Task 7 (`TestGenerateExitInspection_WithoutStructuredEntryInspectionSucceeds`).

---

## File Structure Overview (Modular Monolith)

```text
.
├── Makefile                                                 # Primary DX entry point (build, test, lint, migrate, dev, docker, tf)
├── go.mod
├── cmd/
│   └── gestion-immo/
│       └── main.go                                          # urfave/cli/v3 binary entry point & module composition root
├── deploy/
│   ├── Dockerfile
│   ├── docker-compose.yml                                   # Go monolith, Postgres 16, cloudflared, Gotenberg, pg_dump R2 backup
│   └── scripts/
│       └── backup-to-r2.sh
├── infra/
│   └── terraform/
│       ├── backend.tf                                       # Cloudflare R2 remote state
│       ├── main.tf                                          # R2 buckets, Cloudflare Tunnel, DNS, Pages project
│       ├── variables.tf
│       └── outputs.tf
├── migrations/
│   ├── 000001_auth_and_users.up.sql
│   ├── 000001_auth_and_users.down.sql
│   ├── 000002_portfolio_hierarchy.up.sql
│   ├── 000002_portfolio_hierarchy.down.sql
│   ├── 000003_leases_and_tenants.up.sql
│   ├── 000003_leases_and_tenants.down.sql
│   ├── 000004_payments_and_cash_journal.up.sql
│   ├── 000004_payments_and_cash_journal.down.sql
│   ├── 000005_templates_inspections_incidents.up.sql
│   ├── 000005_templates_inspections_incidents.down.sql
│   ├── 000006_accounting_and_liasse.up.sql
│   └── 000006_accounting_and_liasse.down.sql
├── internal/
│   ├── platform/                                            # Shared technical foundation (zero domain business rules)
│   │   ├── config/
│   │   │   └── config.go                                    # Strict env-var configuration loader
│   │   ├── money/
│   │   │   └── money.go                                     # AmountCents int64, exact VAT & formatting
│   │   ├── secret/
│   │   │   └── secret.go                                    # SecretString implementing slog.LogValuer ("[REDACTED]")
│   │   ├── postgres/
│   │   │   └── migrator.go                                  # golang-migrate runner + pgxpool factory
│   │   ├── r2/
│   │   │   └── storage.go                                   # Cloudflare R2 S3-compatible client
│   │   ├── templating/
│   │   │   ├── docx_engine.go                               # .docx XML placeholder & table engine
│   │   │   └── pdf_engine.go                                # Gotenberg HTML-to-PDF client
│   │   ├── httpx/
│   │   │   └── response.go                                  # Immutable JSON response & semantic status helpers
│   │   └── mcpserver/
│   │       └── registry.go                                  # Streamable HTTP MCP server & tool dispatcher
│   └── modules/                                             # Self-contained DDD Modules (Bounded Contexts)
│       ├── auth/
│       │   ├── domain/
│       │   │   ├── entity.go                                # User, RefreshTokenRecord, MCPApiToken, Claims
│       │   │   └── repository.go
│       │   ├── application/
│       │   │   └── service.go
│       │   ├── infrastructure/
│       │   │   ├── jwt_service.go                           # Isolated JwtService
│       │   │   ├── password_hasher.go                       # Argon2id & SHA-256 refresh/MCP token hashing
│       │   │   └── postgres_repo.go
│       │   ├── interfaces/
│       │   │   └── http/
│       │   │       ├── handler.go
│       │   │       └── middleware.go
│       │   └── module.go                                    # Constructor-injected module wiring
│       ├── portfolio/                                       # Legal entities, shareholders (2059-F), properties, lots, work stocks
│       │   ├── domain/
│       │   │   ├── entity.go
│       │   │   └── repository.go
│       │   ├── application/
│       │   │   └── service.go
│       │   ├── infrastructure/
│       │   │   └── postgres_repo.go
│       │   ├── interfaces/
│       │   │   ├── http/
│       │   │   │   └── handler.go
│       │   │   └── mcp/
│       │   │       └── tools.go
│       │   └── module.go
│       ├── lease/                                           # Tenants (active/archived), guarantors, leases, VAT, rent calls
│       │   ├── domain/
│       │   │   ├── entity.go
│       │   │   └── repository.go
│       │   ├── application/
│       │   │   └── service.go
│       │   ├── infrastructure/
│       │   │   └── postgres_repo.go
│       │   ├── interfaces/
│       │   │   ├── http/
│       │   │   │   └── handler.go
│       │   │   └── mcp/
│       │   │       └── tools.go
│       │   └── module.go
│       ├── payment/                                         # Cash journal 530, CSV/OFX, optional DSP2, FIFO vs month allocation
│       │   ├── domain/
│       │   │   ├── entity.go
│       │   │   ├── allocation_service.go
│       │   │   └── repository.go
│       │   ├── application/
│       │   │   └── service.go
│       │   ├── infrastructure/
│       │   │   ├── csv_importer.go
│       │   │   ├── dsp2_provider.go
│       │   │   └── postgres_repo.go
│       │   ├── interfaces/
│       │   │   ├── http/
│       │   │   │   └── handler.go
│       │   │   └── mcp/
│       │   │       └── tools.go
│       │   └── module.go
│       ├── document/                                        # Templates, rent receipts, 1-click GLI claim dossier builder
│       │   ├── domain/
│       │   │   ├── entity.go
│       │   │   └── repository.go
│       │   ├── application/
│       │   │   └── service.go
│       │   ├── infrastructure/
│       │   │   └── postgres_repo.go
│       │   ├── interfaces/
│       │   │   ├── http/
│       │   │   │   └── handler.go
│       │   │   └── mcp/
│       │   │       └── tools.go
│       │   └── module.go
│       ├── inspection/                                      # Entry/Exit inspections (UI templates, R2 photos) & Incidents
│       │   ├── domain/
│       │   │   ├── inspection.go
│       │   │   ├── incident.go
│       │   │   └── repository.go
│       │   ├── application/
│       │   │   └── service.go
│       │   ├── infrastructure/
│       │   │   └── postgres_repo.go
│       │   ├── interfaces/
│       │   │   ├── http/
│       │   │   │   └── handler.go
│       │   │   └── mcp/
│       │   │       └── tools.go
│       │   └── module.go
│       └── accounting/                                      # PCG ledger, dynamic line toggling calculator, FEC, Liasse PDF
│           ├── domain/
│           │   ├── entity.go
│           │   ├── statement_calculator.go
│           │   └── repository.go
│           ├── application/
│           │   └── service.go
│           ├── infrastructure/
│           │   ├── fec_exporter.go
│           │   ├── liasse_pdf_renderer.go
│           │   └── postgres_repo.go
│           ├── interfaces/
│           │   ├── http/
│           │   │   └── handler.go
│           │   └── mcp/
│           │       └── tools.go
│           └── module.go
└── web/                                                     # SvelteKit SSR + shadcn-svelte frontend
    ├── package.json
    ├── svelte.config.js                                     # @sveltejs/adapter-cloudflare
    └── src/
        ├── lib/
        │   ├── api/client.ts
        │   ├── api/statement-state.ts
        │   └── components/ui/                               # shadcn-svelte components
        └── routes/
            ├── +layout.server.ts
            ├── login/+page.svelte
            ├── entities/[id]/+page.svelte
            ├── payments/negative-positions/+page.svelte
            ├── inspections/[id]/+page.svelte
            └── accounting/[entityId]/statements/+page.svelte
```

---

### Task 1: DX `Makefile`, Platform Foundation (`config`, `money`, `secret`, `postgres`), Terraform & Docker Compose

**Files:**
- Create: `Makefile`
- Create: `go.mod`
- Create: `internal/platform/money/money.go`
- Create: `internal/platform/money/money_test.go`
- Create: `internal/platform/secret/secret.go`
- Create: `internal/platform/config/config.go`
- Create: `internal/platform/config/config_test.go`
- Create: `internal/platform/postgres/migrator.go`
- Create: `internal/platform/httpx/response.go`
- Create: `infra/terraform/backend.tf`
- Create: `infra/terraform/variables.tf`
- Create: `infra/terraform/main.tf`
- Create: `infra/terraform/outputs.tf`
- Create: `deploy/Dockerfile`
- Create: `deploy/docker-compose.yml`
- Create: `deploy/scripts/backup-to-r2.sh`

**Interfaces:**
- Consumes: Environment variables (`DATABASE_URL`, `JWT_SECRET`, `R2_ENDPOINT`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET_NAME`, `GOTENBERG_URL`, `ENABLE_OPEN_BANKING`, `PORT`).
- Produces:
  - `Makefile` targets: `help`, `build`, `dev`, `test`, `test-race`, `lint`, `migrate-up`, `migrate-down`, `migrate-create`, `docker-up`, `docker-down`, `web-dev`, `web-check`, `tf-init`, `tf-plan`, `tf-apply`
  - `money.AmountCents` (`int64`) with `money.ComputeVATCents(ht money.AmountCents, rateBasisPoints int64) money.AmountCents`
  - `secret.String` implementing `slog.LogValuer` (always returns `slog.StringValue("[REDACTED]")`)
  - `config.Config` (immutable struct) and `config.LoadFromEnv(lookup func(string) (string, bool)) (config.Config, error)`
  - `postgres.RunMigrations(dbURL string, migrationsPath string, direction string) error`

- [ ] **Step 1: Initialize `go.mod`, `Makefile`, and write failing tests for `money.AmountCents`, `secret.String`, and `config.LoadFromEnv`**

```go
// internal/platform/config/config_test.go
package config_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/config"
)

func TestLoadFromEnv_MissingRequiredSecretFailsAndRedactsLogs(t *testing.T) {
	_, err := config.LoadFromEnv(func(k string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("expected error when required env vars are missing")
	}

	env := map[string]string{
		"DATABASE_URL":         "postgres://user:supersecretdbpass@postgres:5432/immo?sslmode=disable",
		"JWT_SECRET":           "ultra-secret-jwt-signing-key-32bytes!",
		"R2_ENDPOINT":          "https://acc.r2.cloudflarestorage.com",
		"R2_ACCESS_KEY_ID":     "AKIA123",
		"R2_SECRET_ACCESS_KEY": "r2-secret-value",
		"R2_BUCKET_NAME":       "gestion-immo-files",
		"GOTENBERG_URL":        "http://gotenberg:3000",
		"PORT":                 "8080",
	}
	cfg, err := config.LoadFromEnv(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.EnableOpenBanking {
		t.Fatal("expected EnableOpenBanking to default to false")
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("loaded config", "jwt_secret", cfg.JWTSecret, "r2_secret", cfg.R2SecretAccessKey)
	if strings.Contains(buf.String(), "ultra-secret-jwt-signing-key-32bytes!") || strings.Contains(buf.String(), "r2-secret-value") {
		t.Fatalf("secret leaked in logs: %s", buf.String())
	}
}
```

- [ ] **Step 2: Run test via `make test` to verify it fails**

Run: `make test`
Expected: FAIL with missing packages `internal/platform/config` and `internal/platform/money`.

- [ ] **Step 3: Implement `Makefile`, `money.AmountCents`, `secret.String`, `config.LoadFromEnv`, `postgres.RunMigrations`, `httpx/response.go`, Terraform files, and `deploy/docker-compose.yml`**

Implement `secret.String` with `Value() string` and `LogValue() slog.Value` returning `slog.StringValue("[REDACTED]")`. Implement `money.ComputeVATCents(ht AmountCents, rateBasisPoints int64) AmountCents` using integer arithmetic `(ht * rateBasisPoints + 5000) / 10000`. Configure `Makefile` with self-documented `.PHONY` targets invoking `go test ./...`, `go run ./cmd/gestion-immo`, `docker compose -f deploy/docker-compose.yml`, and `terraform -chdir=infra/terraform`.

- [ ] **Step 4: Run `make test` to verify it passes**

Run: `make test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add Makefile go.mod go.sum internal/platform infra/ deploy/
git commit -m "feat(platform,dx): add Makefile, platform config/money/secret packages, terraform, and docker-compose"
```

---

### Task 2: `auth` Module — Signup with `SUPER_ADMIN` Approval Gate, Isolated `JwtService`, Argon2id Hashing & HTTP Middleware

**Files:**
- Create: `migrations/000001_auth_and_users.up.sql`
- Create: `migrations/000001_auth_and_users.down.sql`
- Create: `internal/modules/auth/domain/entity.go`
- Create: `internal/modules/auth/domain/repository.go`
- Create: `internal/modules/auth/infrastructure/password_hasher.go`
- Create: `internal/modules/auth/infrastructure/jwt_service.go`
- Create: `internal/modules/auth/infrastructure/security_test.go`
- Create: `internal/modules/auth/infrastructure/postgres_repo.go`
- Create: `internal/modules/auth/application/service.go`
- Create: `internal/modules/auth/application/service_test.go`
- Create: `internal/modules/auth/interfaces/http/handler.go`
- Create: `internal/modules/auth/interfaces/http/middleware.go`
- Create: `internal/modules/auth/module.go`

**Interfaces:**
- Consumes: `secret.String`, `httpx`
- Produces:
  - `domain.UserRole` (`SUPER_ADMIN`, `MANAGER`) and `domain.UserStatus` (`PENDING_APPROVAL`, `APPROVED`, `REJECTED`)
  - `infrastructure.JwtService` (`NewJwtService(secret secret.String, issuer string, accessTTL time.Duration) (*JwtService, error)`, `GenerateAccessToken(userID string, email string, role domain.UserRole) (secret.String, error)`, `ValidateAccessToken(token string) (*domain.Claims, error)`)
  - `infrastructure.PasswordHasher` (`NewArgon2Hasher() *PasswordHasher`, `HashPassword(plain secret.String) (string, error)`, `VerifyPassword(plain secret.String, encodedHash string) (bool, error)`, `HashToken(rawToken secret.String) string`)
  - `application.Service` (`BootstrapSuperAdmin(ctx, email string, password secret.String) (UserDTO, error)`, `Signup(ctx, SignupCommand) (UserDTO, error)`, `ApproveUser(ctx, actorUserID string, targetUserID string) (UserDTO, error)`, `RejectUser(ctx, actorUserID string, targetUserID string) (UserDTO, error)`, `ListPendingUsers(ctx, actorUserID string) ([]UserDTO, error)`, `Login(ctx, LoginCommand) (AuthTokensDTO, error)`, `Refresh(ctx, rawRefreshToken secret.String) (AuthTokensDTO, error)`, `CreateMCPToken(ctx, userID string, label string) (secret.String, error)`, `ValidateMCPToken(ctx, rawBearer string) (*domain.User, error)`)
  - `auth.Module` (`NewModule(repo domain.Repository, jwtSvc *infrastructure.JwtService, hasher *infrastructure.PasswordHasher) *Module`, `RegisterHTTPRoutes(mux *http.ServeMux)`)

- [ ] **Step 1: Write failing security, secret redaction, and `SUPER_ADMIN` signup approval gate tests (`TestAuthSecurity_NeverLogsOrStoresPlaintextSecrets` and `TestSignup_RequiresSuperAdminApprovalBeforeLogin`)**

```go
// internal/modules/auth/application/service_test.go
package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

func TestSignup_RequiresSuperAdminApprovalBeforeLogin(t *testing.T) {
	repo := application.NewInMemoryRepo()
	hasher := infrastructure.NewArgon2Hasher()
	jwtSvc, _ := infrastructure.NewJwtService(secret.New("0123456789abcdef0123456789abcdef"), "gestion-immo", 15*time.Minute)
	svc := application.NewService(repo, jwtSvc, hasher)
	ctx := context.Background()

	// 1. Bootstrap Super Admin (you)
	superAdmin, err := svc.BootstrapSuperAdmin(ctx, "yann@investissimo.fr", secret.New("SuperAdminPass123!"))
	if err != nil || superAdmin.Role != domain.RoleSuperAdmin || superAdmin.Status != domain.StatusApproved {
		t.Fatalf("expected approved SUPER_ADMIN, got %+v (err=%v)", superAdmin, err)
	}

	// 2. New user signs up -> status is PENDING_APPROVAL
	applicant, err := svc.Signup(ctx, application.SignupCommand{
		FullName: "Collaborateur Immo",
		Email:    "collab@investissimo.fr",
		Password: secret.New("CollabPass123!"),
	})
	if err != nil || applicant.Status != domain.StatusPendingApproval || applicant.Role != domain.RoleManager {
		t.Fatalf("expected PENDING_APPROVAL MANAGER, got %+v (err=%v)", applicant, err)
	}

	// 3. Login before approval MUST fail with ErrAccountPendingApproval
	_, err = svc.Login(ctx, application.LoginCommand{
		Email:    "collab@investissimo.fr",
		Password: secret.New("CollabPass123!"),
	})
	if !errors.Is(err, domain.ErrAccountNotApproved) {
		t.Fatalf("expected ErrAccountNotApproved when logging in before approval, got %v", err)
	}

	// 4. Non-super-admin cannot approve users
	_, err = svc.ApproveUser(ctx, applicant.ID, applicant.ID)
	if !errors.Is(err, domain.ErrForbiddenSuperAdminOnly) {
		t.Fatalf("expected ErrForbiddenSuperAdminOnly, got %v", err)
	}

	// 5. Super Admin approves the user -> Login now succeeds
	approved, err := svc.ApproveUser(ctx, superAdmin.ID, applicant.ID)
	if err != nil || approved.Status != domain.StatusApproved {
		t.Fatalf("expected APPROVED status after super admin approval, got %+v (err=%v)", approved, err)
	}
	tokens, err := svc.Login(ctx, application.LoginCommand{
		Email:    "collab@investissimo.fr",
		Password: secret.New("CollabPass123!"),
	})
	if err != nil || tokens.AccessToken.Value() == "" {
		t.Fatalf("expected login to succeed after approval, got err=%v", err)
	}
}
```

```go
// internal/modules/auth/infrastructure/security_test.go
package infrastructure_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

func TestAuthSecurity_NeverLogsOrStoresPlaintextSecrets(t *testing.T) {
	hasher := infrastructure.NewArgon2Hasher()
	rawPass := secret.New("MySuperSecretPassword123!")
	passHash, err := hasher.HashPassword(rawPass)
	if err != nil || passHash == rawPass.Value() || strings.Contains(passHash, rawPass.Value()) {
		t.Fatalf("password hash invalid or plaintext: err=%v", err)
	}
	rawRefresh := secret.New("rt_live_9876543210abcdef")
	if hasher.HashToken(rawRefresh) == rawRefresh.Value() {
		t.Fatal("refresh token stored in plaintext")
	}
	jwtSvc, _ := infrastructure.NewJwtService(secret.New("0123456789abcdef0123456789abcdef"), "gestion-immo", 15*time.Minute)
	jwtTok, _ := jwtSvc.GenerateAccessToken("usr_1", "yann@investissimo.fr", domain.RoleSuperAdmin)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	logger.Info("auth event", "password", rawPass, "jwt", jwtTok, "refresh_token", rawRefresh)
	for _, s := range []string{rawPass.Value(), jwtTok.Value(), rawRefresh.Value()} {
		if strings.Contains(logBuf.String(), s) {
			t.Fatalf("plaintext secret leaked in log output: %s", logBuf.String())
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/auth/... -v`
Expected: FAIL with undefined package `internal/modules/auth/application`.

- [ ] **Step 3: Implement `000001_auth_and_users` migration, `PasswordHasher`, isolated `JwtService`, `auth/application/service.go` (Signup, SuperAdmin approval gate, Login), HTTP handlers/middleware (`RequireAuth`, `RequireSuperAdmin`), and `auth/module.go`**

Wire HTTP endpoints in `auth/module.go`:
- `POST /api/v1/auth/signup` -> returns `202 Accepted` with user in `PENDING_APPROVAL` status (no tokens issued).
- `POST /api/v1/auth/login` -> returns `403 Forbidden` if user status is `PENDING_APPROVAL` or `REJECTED`; `200 OK` if `APPROVED`.
- `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`, `POST /api/v1/auth/mcp-tokens`.
- `GET /api/v1/admin/users/pending`, `POST /api/v1/admin/users/{id}/approve`, `POST /api/v1/admin/users/{id}/reject` -> protected by `RequireSuperAdmin` middleware (`403 Forbidden` for non-`SUPER_ADMIN`).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/auth/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add migrations/000001* internal/modules/auth
git commit -m "feat(auth): add signup with SUPER_ADMIN approval gate, isolated JwtService, and Argon2id/SHA-256 hashing"
```

---

### Task 3: `portfolio` Module — Legal Entities, Associates (`2059-F`), Properties & Lots (`Immobilisation` vs `Marchandise` + `Stock travaux`)

**Files:**
- Create: `migrations/000002_portfolio_hierarchy.up.sql`
- Create: `migrations/000002_portfolio_hierarchy.down.sql`
- Create: `internal/modules/portfolio/domain/entity.go`
- Create: `internal/modules/portfolio/domain/repository.go`
- Create: `internal/modules/portfolio/application/service.go`
- Create: `internal/modules/portfolio/application/service_test.go`
- Create: `internal/modules/portfolio/infrastructure/postgres_repo.go`
- Create: `internal/modules/portfolio/interfaces/http/handler.go`
- Create: `internal/modules/portfolio/interfaces/mcp/tools.go`
- Create: `internal/modules/portfolio/module.go`

**Interfaces:**
- Consumes: `money.AmountCents`, `mcpserver.Registry`
- Produces:
  - `domain.EntityType` (`SCI_IR`, `SCI_IS`, `SARL_IS`, `SARL_FAMILLE`, `SAS`, `NOM_PROPRE`, `INDIVISION`)
  - `domain.AccountingCategory` (`IMMOBILISATION`, `MARCHANDISE`, `PRODUIT_FINI`)
  - `application.Service` (`CreateLegalEntity(ctx, CreateLegalEntityCmd) (LegalEntityDTO, error)`, `UpsertShareholder(ctx, UpsertShareholderCmd) (ShareholderDTO, error)`, `CreateProperty(ctx, CreatePropertyCmd) (PropertyDTO, error)`, `CreateLot(ctx, CreateLotCmd) (LotDTO, error)`, `GetEntityPortfolioSummary(ctx, entityID string) (EntityPortfolioSummaryDTO, error)`)
  - `portfolio.Module` (`NewModule(repo domain.Repository) *Module`, `RegisterHTTPRoutes(mux *http.ServeMux)`, `RegisterMCPTools(reg *mcpserver.Registry)`)

- [ ] **Step 1: Write failing tests for `LegalEntity`, `Shareholder` ownership percentage (`2059-F`), and `Property` accounting classification (`Marchandise` + `Stock travaux`)**

```go
// internal/modules/portfolio/application/service_test.go
package application_test

import (
	"context"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func TestCreateEntityAndMerchantOfGoodsPropertyWithWorkStock(t *testing.T) {
	svc := application.NewService(application.NewInMemoryRepo())
	ctx := context.Background()

	ent, err := svc.CreateLegalEntity(ctx, application.CreateLegalEntityCmd{
		Name:              "SARL INVESTISSIMO",
		EntityType:        "SARL_IS",
		SIRET:             "40193236300021",
		NAFCode:           "6810Z",
		Address:           "3 RUE LAFAYETTE, 13001 MARSEILLE",
		ShareCapitalCents: money.AmountCents(196278_11),
		TotalShares:       12875,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sh, err := svc.UpsertShareholder(ctx, application.UpsertShareholderCmd{
		LegalEntityID:         ent.ID,
		FullName:              "AMSELLEM MAURICE",
		SharesCount:           6375,
		CurrentAccountCents:   money.AmountCents(404032_00),
		RemunerationPaidCents: money.AmountCents(10296_00),
	})
	if err != nil {
		t.Fatalf("unexpected shareholder error: %v", err)
	}
	if sh.DetentionBasisPoints != 4951 { // 6375 / 12875 = 49.51%
		t.Fatalf("expected 4951 bps (49.51%%), got %d", sh.DetentionBasisPoints)
	}

	prop, err := svc.CreateProperty(ctx, application.CreatePropertyCmd{
		LegalEntityID:      ent.ID,
		Label:              "APPART RUE SAINTE MARSEILLE 1er",
		PropertyType:       "APPARTEMENT",
		AccountingCategory: "MARCHANDISE",
		AcquisitionCents:   money.AmountCents(24284_00),
		WorkStockCents:     money.AmountCents(39835_00),
	})
	if err != nil || prop.TotalBookValueCents != money.AmountCents(64119_00) {
		t.Fatalf("expected TotalBookValueCents 64119_00, got %+v (err=%v)", prop, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/portfolio/... -v`
Expected: FAIL with undefined package `internal/modules/portfolio/application`.

- [ ] **Step 3: Implement `000002_portfolio_hierarchy` migration, `portfolio` domain, application service, Postgres repo, HTTP handler, MCP tools (`create_legal_entity`, `upsert_shareholder`, `list_entities`, `get_entity_summary`, `create_property`, `create_lot`, `list_properties`), and `portfolio/module.go`**

Register REST endpoints in `portfolio/module.go`: `POST /api/v1/entities`, `GET /api/v1/entities`, `GET /api/v1/entities/{id}`, `POST /api/v1/entities/{id}/shareholders`, `POST /api/v1/entities/{id}/properties`, `POST /api/v1/properties/{id}/lots`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/portfolio/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add migrations/000002* internal/modules/portfolio
git commit -m "feat(portfolio): add portfolio module with legal entities, 2059-F shareholders, properties, lots, and MCP tools"
```

---

### Task 4: `lease` Module — Tenants (Present & Archived), Leases (Residential vs Commercial VAT), GLI Metadata & Rent Calls

**Files:**
- Create: `migrations/000003_leases_and_tenants.up.sql`
- Create: `migrations/000003_leases_and_tenants.down.sql`
- Create: `internal/modules/lease/domain/entity.go`
- Create: `internal/modules/lease/domain/repository.go`
- Create: `internal/modules/lease/application/service.go`
- Create: `internal/modules/lease/application/service_test.go`
- Create: `internal/modules/lease/infrastructure/postgres_repo.go`
- Create: `internal/modules/lease/interfaces/http/handler.go`
- Create: `internal/modules/lease/interfaces/mcp/tools.go`
- Create: `internal/modules/lease/module.go`

**Interfaces:**
- Consumes: `money.AmountCents`, `money.ComputeVATCents`, `mcpserver.Registry`
- Produces:
  - `domain.LeaseType` (`HABITATION_VIDE`, `HABITATION_MEUBLE`, `COMMERCIAL_3_6_9`, `PROFESSIONNEL`, `PARKING`, `PRECAIRE`)
  - `domain.RentCallStatus` (`ATTENDU`, `PARTIEL`, `PAYE`, `IMPAYE`)
  - `domain.RentCall` (immutable struct)
  - `application.Service` (`CreateTenant(ctx, CreateTenantCmd) (TenantDTO, error)`, `ArchiveTenant(ctx, tenantID string) error`, `CreateLease(ctx, CreateLeaseCmd) (LeaseDTO, error)`, `GenerateMonthlyRentCalls(ctx, entityID string, periodMonth string) ([]RentCallDTO, error)`)
  - `lease.Module` (`NewModule(repo domain.Repository) *Module`, `RegisterHTTPRoutes(mux *http.ServeMux)`, `RegisterMCPTools(reg *mcpserver.Registry)`)

- [ ] **Step 1: Write failing tests for residential vs commercial lease rent calls (`TestCreateCommercialRentCall_VATExactCentsNoFloatDrift`)**

```go
// internal/modules/lease/application/service_test.go
package application_test

import (
	"context"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func TestCreateCommercialRentCall_VATExactCentsNoFloatDrift(t *testing.T) {
	svc := application.NewService(application.NewInMemoryRepo())
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, application.CreateTenantCmd{
		FullName: "BOULANGERIE DU VIEUX PORT",
		Email:    "contact@boulangerie.fr",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateLease(ctx, application.CreateLeaseCmd{
		LegalEntityID:        "ent_1",
		LotID:                "lot_comm_1",
		TenantID:             tenant.ID,
		LeaseType:            "COMMERCIAL_3_6_9",
		RentNetCents:         money.AmountCents(1912_67),
		ChargesCents:         money.AmountCents(796_50),
		SubjectToVAT:         true,
		VATRateBasisPoints:   2000, // 20.00%
		SecurityDepositCents: money.AmountCents(3825_34),
		DueDayOfMonth:        5,
		GLIPolicyNumber:      "GLI-AXA-99881",
	})
	if err != nil {
		t.Fatal(err)
	}

	calls, err := svc.GenerateMonthlyRentCalls(ctx, "ent_1", "2026-01")
	if err != nil || len(calls) != 1 {
		t.Fatalf("expected 1 rent call, got %d (err=%v)", len(calls), err)
	}
	rc := calls[0]
	// VAT 20% on Rent (1912.67) = 382.534 -> 382_53 cents; Total TTC = 1912_67 + 382_53 + 796_50 = 3091_70
	if rc.VATCents != money.AmountCents(382_53) || rc.TotalDueCents != money.AmountCents(3091_70) || rc.RemainingDueCents != money.AmountCents(3091_70) {
		t.Fatalf("unexpected rent call amounts: %+v", rc)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/lease/... -v`
Expected: FAIL with undefined package `internal/modules/lease/application`.

- [ ] **Step 3: Implement `000003_leases_and_tenants` migration, `lease` domain, application service, Postgres repo, HTTP handler, MCP tools (`create_tenant`, `create_lease`, `list_leases`), and `lease/module.go`**

Expose REST endpoints: `POST /api/v1/tenants`, `GET /api/v1/tenants`, `POST /api/v1/leases`, `GET /api/v1/leases`, `POST /api/v1/entities/{id}/rent-calls/generate`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/lease/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add migrations/000003* internal/modules/lease
git commit -m "feat(lease): add lease module with tenants, commercial VAT rent calls, GLI metadata, and MCP tools"
```

---

### Task 5: `payment` Module — Cash Journal (`530`), CSV/OFX Batch, Optional DSP2, FIFO vs Explicit-Month Allocation & Negative Positions

**Files:**
- Create: `migrations/000004_payments_and_cash_journal.up.sql`
- Create: `migrations/000004_payments_and_cash_journal.down.sql`
- Create: `internal/modules/payment/domain/entity.go`
- Create: `internal/modules/payment/domain/allocation_service.go`
- Create: `internal/modules/payment/domain/allocation_service_test.go`
- Create: `internal/modules/payment/domain/repository.go`
- Create: `internal/modules/payment/application/service.go`
- Create: `internal/modules/payment/infrastructure/csv_importer.go`
- Create: `internal/modules/payment/infrastructure/dsp2_provider.go`
- Create: `internal/modules/payment/infrastructure/postgres_repo.go`
- Create: `internal/modules/payment/interfaces/http/handler.go`
- Create: `internal/modules/payment/interfaces/mcp/tools.go`
- Create: `internal/modules/payment/module.go`

**Interfaces:**
- Consumes: `lease/domain.RentCall`, `money.AmountCents`, `mcpserver.Registry`
- Produces:
  - `domain.AllocationService` (`DetectTargetMonth(label string, fallbackYear int) (string, bool)`, `AllocatePayment(unpaidCalls []leasedomain.RentCall, amountCents money.AmountCents, label string, paymentDate time.Time) ([]domain.RentAllocation, []leasedomain.RentCall)`)
  - `application.Service` (`RecordCashPayment(ctx, RecordCashPaymentCmd) (PaymentReceiptDTO, error)`, `RecordBankPayment(ctx, RecordBankPaymentCmd) (PaymentReceiptDTO, error)`, `ImportPaymentsCSV(ctx, entityID string, r io.Reader) (BatchImportResultDTO, error)`, `ListNegativePositions(ctx, entityID string) ([]NegativePositionDTO, error)`)
  - `payment.Module` (`NewModule(...) *Module`, `RegisterHTTPRoutes(mux *http.ServeMux)`, `RegisterMCPTools(reg *mcpserver.Registry)`)

- [ ] **Step 1: Write failing tests for explicit month vs FIFO allocation and overpayment spillover (`TestAllocatePayment_ExplicitMonthWithOverpaymentSpillsToFIFO`)**

```go
// internal/modules/payment/domain/allocation_service_test.go
package domain_test

import (
	"testing"
	"time"

	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func TestAllocatePayment_FIFOWhenNoMonthSpecified(t *testing.T) {
	svc := domain.NewAllocationService()
	unpaid := []leasedomain.RentCall{
		{ID: "rc_jan", PeriodMonth: "2026-01", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusImpaye},
		{ID: "rc_feb", PeriodMonth: "2026-02", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusImpaye},
		{ID: "rc_mar", PeriodMonth: "2026-03", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusImpaye},
	}

	allocs, updated := svc.AllocatePayment(unpaid, money.AmountCents(800_00), "VIR SEPA M DUPONT", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	if len(allocs) != 1 || allocs[0].RentCallID != "rc_jan" || allocs[0].AppliedCents != 800_00 {
		t.Fatalf("expected FIFO allocation to rc_jan, got %+v", allocs)
	}
	if updated[0].Status != leasedomain.StatusPaye || updated[1].Status != leasedomain.StatusImpaye || updated[2].Status != leasedomain.StatusImpaye {
		t.Fatalf("expected Jan PAYE and Feb/Mar IMPAYE, got %+v", updated)
	}
}

func TestAllocatePayment_ExplicitMonthWithOverpaymentSpillsToFIFO(t *testing.T) {
	svc := domain.NewAllocationService()
	unpaid := []leasedomain.RentCall{
		{ID: "rc_jan", PeriodMonth: "2026-01", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusImpaye},
		{ID: "rc_feb", PeriodMonth: "2026-02", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusImpaye},
		{ID: "rc_mar", PeriodMonth: "2026-03", TotalDueCents: 800_00, RemainingDueCents: 800_00, Status: leasedomain.StatusImpaye},
	}

	// Pays 800 EUR specifying "mars" -> March is paid, Jan & Feb remain unpaid
	allocs, updated := svc.AllocatePayment(unpaid, money.AmountCents(800_00), "Virement loyer mars", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	if len(allocs) != 1 || allocs[0].RentCallID != "rc_mar" {
		t.Fatalf("expected allocation to rc_mar, got %+v", allocs)
	}
	if updated[2].Status != leasedomain.StatusPaye || updated[0].Status != leasedomain.StatusImpaye || updated[1].Status != leasedomain.StatusImpaye {
		t.Fatalf("expected Mar PAYE and Jan/Feb IMPAYE, got %+v", updated)
	}

	// Overpayment case: pays 1000 EUR specifying "Loyer Mars 2026" -> 800 to Mar (PAYE), remaining 200 to Jan via FIFO (PARTIEL, 600 remaining)
	allocs2, updated2 := svc.AllocatePayment(unpaid, money.AmountCents(1000_00), "Virement Loyer Mars 2026", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	if len(allocs2) != 2 || allocs2[0].RentCallID != "rc_mar" || allocs2[1].RentCallID != "rc_jan" || allocs2[1].AppliedCents != 200_00 {
		t.Fatalf("expected Mar then Jan spillover, got %+v", allocs2)
	}
	if updated2[2].Status != leasedomain.StatusPaye || updated2[0].Status != leasedomain.StatusPartiel || updated2[0].RemainingDueCents != 600_00 {
		t.Fatalf("unexpected statuses after spillover: %+v", updated2)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/payment/... -v`
Expected: FAIL with undefined package `internal/modules/payment/domain`.

- [ ] **Step 3: Implement `AllocationService` (French month names `janvier`..`décembre` + `MM/YYYY` parser + FIFO fallback), `000004_payments_and_cash_journal` migration, Cash Journal (`530`), CSV importer, optional `DSP2Provider`, HTTP handlers, MCP tools (`record_cash_payment`, `record_bank_payment`, `list_negative_positions`), and `payment/module.go`**

Expose REST endpoints: `POST /api/v1/payments/cash`, `POST /api/v1/payments/bank`, `POST /api/v1/payments/import-csv`, `GET /api/v1/entities/{id}/negative-positions`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/payment/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add migrations/000004* internal/modules/payment
git commit -m "feat(payment): add payment module with explicit-month vs FIFO allocation, cash journal, and negative positions"
```

---

### Task 6: Platform R2/Templating & `document` Module — Leases, Receipts & 1-Click GLI Claim Dossier

**Files:**
- Create: `migrations/000005_templates_inspections_incidents.up.sql`
- Create: `migrations/000005_templates_inspections_incidents.down.sql`
- Create: `internal/platform/r2/storage.go`
- Create: `internal/platform/templating/docx_engine.go`
- Create: `internal/platform/templating/docx_engine_test.go`
- Create: `internal/platform/templating/pdf_engine.go`
- Create: `internal/modules/document/domain/entity.go`
- Create: `internal/modules/document/domain/repository.go`
- Create: `internal/modules/document/application/service.go`
- Create: `internal/modules/document/infrastructure/postgres_repo.go`
- Create: `internal/modules/document/interfaces/http/handler.go`
- Create: `internal/modules/document/interfaces/mcp/tools.go`
- Create: `internal/modules/document/module.go`

**Interfaces:**
- Consumes: `portfolio/application.Service`, `lease/application.Service`, `payment/application.Service`
- Produces:
  - `r2.ObjectStorage` (`Upload(ctx, key string, contentType string, body io.Reader) error`, `Download(ctx, key string) (io.ReadCloser, error)`, `PresignGetURL(ctx, key string, ttl time.Duration) (string, error)`)
  - `templating.DocxEngine` (`RenderDocx(templateBytes []byte, vars map[string]string, tables map[string][]map[string]string) ([]byte, error)`)
  - `document/application.Service` (`GenerateFromTemplate(ctx, GenerateDocCmd) (StoredFileDTO, error)`, `AssembleGLIDossier(ctx, leaseID string, templateID string) (GLIDossierArchiveDTO, error)`)
  - `document.Module` (`NewModule(...) *Module`, `RegisterHTTPRoutes(mux *http.ServeMux)`, `RegisterMCPTools(reg *mcpserver.Registry)`)

- [ ] **Step 1: Write failing test for `.docx` XML placeholder interpolation and GLI unpaid rent table injection**

```go
// internal/platform/templating/docx_engine_test.go
package templating_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

func TestRenderDocx_ReplacesPlaceholdersAndUnpaidMonthsTable(t *testing.T) {
	engine := templating.NewDocxEngine()
	sampleDocx := templating.BuildMinimalTestDocx(t, "Bailleur: {{societe.nom}} - Locataire: {{locataire.nom}} - Total impayé: {{impaye.total}}")

	out, err := engine.RenderDocx(sampleDocx, map[string]string{
		"societe.nom":   "SARL INVESTISSIMO",
		"locataire.nom": "Jean Dupont",
		"impaye.total":  "1 600,00 €",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected docx error: %v", err)
	}
	extracted := templating.ExtractTextFromDocx(t, bytes.NewReader(out))
	if !strings.Contains(extracted, "SARL INVESTISSIMO") || !strings.Contains(extracted, "1 600,00 €") {
		t.Fatalf("expected replaced variables in docx, got: %s", extracted)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/templating/... -v`
Expected: FAIL with undefined package `internal/platform/templating`.

- [ ] **Step 3: Implement `r2.Storage`, `DocxEngine`, `PdfEngine`, and `document` module (`GenerateFromTemplate`, `AssembleGLIDossier`, HTTP handlers, and MCP tools `generate_document_from_template`, `generate_gli_dossier`)**

`AssembleGLIDossier` populates the GLI template with entity, property, lease, tenant, and unpaid rent table data, renders the tenant ledger PDF, pulls lease/EDL attachments from R2, and saves the complete ZIP package to R2.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/platform/templating/... ./internal/modules/document/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add migrations/000005* internal/platform/r2 internal/platform/templating internal/modules/document
git commit -m "feat(document): add R2 storage, docx/pdf templating engine, and document/GLI dossier module"
```

---

### Task 7: `inspection` Module — Move-In/Move-Out Inspections (UI Grid Templating & Comparison) & Incidents

**Files:**
- Create: `internal/modules/inspection/domain/inspection.go`
- Create: `internal/modules/inspection/domain/incident.go`
- Create: `internal/modules/inspection/domain/repository.go`
- Create: `internal/modules/inspection/application/service.go`
- Create: `internal/modules/inspection/application/service_test.go`
- Create: `internal/modules/inspection/infrastructure/postgres_repo.go`
- Create: `internal/modules/inspection/interfaces/http/handler.go`
- Create: `internal/modules/inspection/interfaces/mcp/tools.go`
- Create: `internal/modules/inspection/module.go`

**Interfaces:**
- Consumes: `r2.ObjectStorage`, `templating.PdfEngine`, `mcpserver.Registry`
- Produces:
  - `domain.InspectionType` (`ENTRANT`, `SORTANT`)
  - `domain.ConditionState` (`NEUF`, `BON`, `ETAT_D_USAGE`, `MAUVAIS`, `HORS_SERVICE`)
  - `application.Service` (`CreateTemplate(ctx, CreateInspectionTemplateCmd) (InspectionTemplateDTO, error)`, `CreateInspection(ctx, CreateInspectionCmd) (InspectionDTO, error)`, `UpsertInspectionItems(ctx, inspectionID string, items []InspectionItemInput) (InspectionDTO, error)`, `GenerateInspectionReport(ctx, inspectionID string) (InspectionReportDTO, error)`, `CreateIncident(ctx, CreateIncidentCmd) (IncidentDTO, error)`, `UpdateIncident(ctx, UpdateIncidentCmd) (IncidentDTO, error)`)
  - `inspection.Module` (`NewModule(...) *Module`, `RegisterHTTPRoutes(mux *http.ServeMux)`, `RegisterMCPTools(reg *mcpserver.Registry)`)

- [ ] **Step 1: Write failing test for inspection item population and standalone exit inspection (`TestGenerateExitInspection_WithoutStructuredEntryInspectionSucceeds`)**

```go
// internal/modules/inspection/application/service_test.go
package application_test

import (
	"context"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/application"
)

func TestGenerateExitInspection_WithoutStructuredEntryInspectionSucceeds(t *testing.T) {
	svc := application.NewService(application.NewInMemoryRepo())
	ctx := context.Background()

	tpl, err := svc.CreateTemplate(ctx, application.CreateInspectionTemplateCmd{
		Name:  "Appartement T2 Standard",
		Rooms: []string{"Séjour", "Cuisine", "Chambre", "Salle de bain"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Exit inspection created for a legacy lease that has NO structured entry inspection (EntryInspectionID == nil)
	exitInsp, err := svc.CreateInspection(ctx, application.CreateInspectionCmd{
		LeaseID:           "lease_legacy_1",
		TemplateID:        tpl.ID,
		Type:              "SORTANT",
		EntryInspectionID: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error creating exit inspection without entry: %v", err)
	}

	updated, err := svc.UpsertInspectionItems(ctx, exitInsp.ID, []application.InspectionItemInput{
		{RoomName: "Séjour", ElementName: "Parquet", Condition: "MAUVAIS", Observation: "Rayures profondes près de la baie vitrée", PhotoR2Keys: []string{"edl/lease_legacy_1/sejour_1.jpg"}},
	})
	if err != nil || len(updated.Items) != 1 {
		t.Fatalf("expected 1 inspection item, got %+v (err=%v)", updated, err)
	}

	report, err := svc.GenerateInspectionReport(ctx, exitInsp.ID)
	if err != nil || report.HasEntryComparison {
		t.Fatalf("expected standalone report with HasEntryComparison=false, got %+v (err=%v)", report, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/inspection/... -v`
Expected: FAIL with undefined package `internal/modules/inspection/application`.

- [ ] **Step 3: Implement `inspection` module (domain, application service, Postgres repo, HTTP handler, MCP tools `get_inspection_template`, `create_inspection`, `upsert_inspection_items`, `create_incident`, `update_incident`, and `inspection/module.go`)**

Expose REST endpoints: `POST /api/v1/inspection-templates`, `POST /api/v1/inspections`, `PUT /api/v1/inspections/{id}/items`, `POST /api/v1/inspections/{id}/pdf`, `POST /api/v1/incidents`, `PATCH /api/v1/incidents/{id}`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/inspection/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/inspection
git commit -m "feat(inspection): add inspection and incident module with UI templates and Antigravity MCP tools"
```

---

### Task 8: `accounting` Module — PCG Ledger, Dynamic Line Toggling (`StatementCalculator`), FEC Export & Liasse Fiscale PDF

**Files:**
- Create: `migrations/000006_accounting_and_liasse.up.sql`
- Create: `migrations/000006_accounting_and_liasse.down.sql`
- Create: `internal/modules/accounting/domain/entity.go`
- Create: `internal/modules/accounting/domain/statement_calculator.go`
- Create: `internal/modules/accounting/domain/statement_calculator_test.go`
- Create: `internal/modules/accounting/domain/repository.go`
- Create: `internal/modules/accounting/application/service.go`
- Create: `internal/modules/accounting/infrastructure/fec_exporter.go`
- Create: `internal/modules/accounting/infrastructure/liasse_pdf_renderer.go`
- Create: `internal/modules/accounting/infrastructure/postgres_repo.go`
- Create: `internal/modules/accounting/interfaces/http/handler.go`
- Create: `internal/modules/accounting/interfaces/mcp/tools.go`
- Create: `internal/modules/accounting/module.go`

**Interfaces:**
- Consumes: `money.AmountCents`, `templating.PdfEngine`, `mcpserver.Registry`
- Produces:
  - `domain.StatementCalculator` (`ComputeStatements(rawLines []domain.AccountBalanceLine, disabledLineIDs map[string]bool) domain.ComputedFinancialStatementsDTO`)
  - `infrastructure.FECExporter` (`ExportFEC(entries []domain.FECEntry) ([]byte, error)`)
  - `application.Service` (`RecordJournalEntry(ctx, RecordJournalEntryCmd) (JournalEntryDTO, error)`, `PreviewFinancialStatements(ctx, entityID string, fiscalYear int, disabledLineIDs []string) (domain.ComputedFinancialStatementsDTO, error)`, `GenerateAnnualAccountsPDF(ctx, entityID string, fiscalYear int, disabledLineIDs []string) (StoredFileDTO, error)`, `ExportEntityFEC(ctx, entityID string, fiscalYear int) ([]byte, error)`)
  - `accounting.Module` (`NewModule(...) *Module`, `RegisterHTTPRoutes(mux *http.ServeMux)`, `RegisterMCPTools(reg *mcpserver.Registry)`)

- [ ] **Step 1: Write failing tests for dynamic line toggling, cascading subtotal/SIG/Passif recalculation, and zero N-1 variation (`TestCalculateStatement_DisabledLineFlipsResultAndZeroPrevYearReturnsNS`)**

```go
// internal/modules/accounting/domain/statement_calculator_test.go
package domain_test

import (
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func TestCalculateStatement_DisabledLineFlipsResultAndZeroPrevYearReturnsNS(t *testing.T) {
	calc := domain.NewStatementCalculator()
	lines := []domain.AccountBalanceLine{
		{LineID: "rev_locaux", AccountCode: "706100", Label: "LOCATIONS LOCAUX", Section: "PRODUITS_EXPLOITATION", NetNCents: money.AmountCents(133607_00), NetNMinus1Cents: money.AmountCents(132604_00)},
		{LineID: "rev_comm", AccountCode: "706200", Label: "LOYERS COMMERCIAUX", Section: "PRODUITS_EXPLOITATION", NetNCents: money.AmountCents(22952_00), NetNMinus1Cents: money.AmountCents(15301_00)},
		{LineID: "chg_loc", AccountCode: "614000", Label: "CHARGES LOCATIVES", Section: "CHARGES_EXTERNES", NetNCents: money.AmountCents(45001_00), NetNMinus1Cents: money.AmountCents(77663_00)},
		{LineID: "chg_tf", AccountCode: "635120", Label: "TAXES FONCIERES", Section: "IMPOTS_TAXES", NetNCents: money.AmountCents(56916_00), NetNMinus1Cents: money.AmountCents(55767_00)},
		{LineID: "chg_fourn_admin", AccountCode: "606400", Label: "FOURNITURES ADMINISTRATIVES", Section: "CHARGES_EXTERNES", NetNCents: money.AmountCents(50000_00), NetNMinus1Cents: 0},
	}

	// 1. All lines active: Total Produits = 156559, Total Charges = 151917 -> Resultat = +4642_00
	full := calc.ComputeStatements(lines, nil)
	if full.TotalProduitsExploitationNCents != 156559_00 || full.ResultatExerciceNCents != 4642_00 {
		t.Fatalf("unexpected full totals: %+v", full)
	}
	var adminLine domain.ComputedLineDTO
	for _, l := range full.Lines {
		if l.LineID == "chg_fourn_admin" {
			adminLine = l
		}
	}
	if adminLine.VariationPctFormatted != "N/S" {
		t.Fatalf("expected N/S when N-1 is 0, got %q", adminLine.VariationPctFormatted)
	}

	// 2. Disable "rev_comm" (22952_00) -> Total Produits drops to 133607_00, Resultat flips to deficit -18310_00
	disabled := map[string]bool{"rev_comm": true}
	filtered := calc.ComputeStatements(lines, disabled)
	if filtered.TotalProduitsExploitationNCents != 133607_00 {
		t.Fatalf("expected TotalProduitsExploitationNCents 133607_00, got %d", filtered.TotalProduitsExploitationNCents)
	}
	if filtered.ResultatExerciceNCents != -18310_00 || filtered.BilanPassifResultatNCents != -18310_00 {
		t.Fatalf("expected ResultatExerciceNCents and BilanPassifResultatNCents -18310_00, got %d / %d", filtered.ResultatExerciceNCents, filtered.BilanPassifResultatNCents)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/accounting/... -v`
Expected: FAIL with undefined package `internal/modules/accounting/domain`.

- [ ] **Step 3: Implement `StatementCalculator`, `000006_accounting_and_liasse` migration, `FECExporter` (18 normative DGFiP columns), `LiassePDFRenderer` (Comptes Annuels SGCAM layout + CERFA `2065`, `2050`–`2059-F`, `1330-CVAE`, `2072`, Détail Actif/Passif/Résultat, SIG), HTTP handlers, MCP tools (`preview_financial_statement`, `generate_annual_accounts_pdf`, `export_fec`), and `accounting/module.go`**

Expose REST endpoints: `POST /api/v1/entities/{id}/journal-entries`, `POST /api/v1/entities/{id}/statements/preview` (accepting `disabled_line_ids`), `POST /api/v1/entities/{id}/statements/pdf`, `GET /api/v1/entities/{id}/fec`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/modules/accounting/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add migrations/000006* internal/modules/accounting
git commit -m "feat(accounting): add accounting module with dynamic line toggling calculator, FEC exporter, and Liasse PDF renderer"
```

---

### Task 9: MCP Registry (`internal/platform/mcpserver`) & `urfave/cli/v3` Binary Entry Point (`cmd/gestion-immo`)

**Files:**
- Create: `internal/platform/mcpserver/registry.go`
- Create: `internal/platform/mcpserver/registry_test.go`
- Create: `cmd/gestion-immo/main.go`
- Create: `cmd/gestion-immo/main_test.go`

**Interfaces:**
- Consumes: `github.com/urfave/cli/v3`, `auth.Module`, `portfolio.Module`, `lease.Module`, `payment.Module`, `document.Module`, `inspection.Module`, `accounting.Module`
- Produces:
  - `mcpserver.Registry` (`NewRegistry(tokenValidator TokenValidatorFunc) *Registry`, `RegisterTool(def ToolDefinition, handler ToolHandlerFunc)`, `ServeHTTP(w http.ResponseWriter, r *http.Request)`) mounted at `POST /mcp` and `GET /mcp`
  - `cmd/gestion-immo` CLI commands via `urfave/cli/v3`:
    - `gestion-immo serve` (starts HTTP API + MCP server)
    - `gestion-immo migrate up` / `gestion-immo migrate down`
    - `gestion-immo admin bootstrap --email <email>` (creates/ensures your `SUPER_ADMIN` account)
    - `gestion-immo admin approve --email <email>` (approves a `PENDING_APPROVAL` signup from CLI)
    - `gestion-immo token create --email <email> --label <label>`

- [ ] **Step 1: Write failing integration test for MCP JSON-RPC registry and `urfave/cli/v3` command wiring**

```go
// internal/platform/mcpserver/registry_test.go
package mcpserver_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

func TestMCPRegistry_RequiresBearerAndExposesAllModuleToolsWithoutMigrateDriveRecord(t *testing.T) {
	srv := mcpserver.NewTestFullAppRegistry(t, "mcp_valid_token_123")

	reqUnauth := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	recUnauth := httptest.NewRecorder()
	srv.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without bearer token, got %d", recUnauth.Code)
	}

	reqList := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	reqList.Header.Set("Authorization", "Bearer mcp_valid_token_123")
	recList := httptest.NewRecorder()
	srv.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recList.Code)
	}
	body := recList.Body.String()
	for _, requiredTool := range []string{"create_legal_entity", "record_cash_payment", "list_negative_positions", "upsert_inspection_items", "generate_gli_dossier", "preview_financial_statement"} {
		if !strings.Contains(body, requiredTool) {
			t.Fatalf("missing required MCP tool %q in %s", requiredTool, body)
		}
	}
	if strings.Contains(body, "migrate_drive_record") {
		t.Fatal("migrate_drive_record must not exist")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `make test`
Expected: FAIL with undefined `NewTestFullAppRegistry` and missing `cmd/gestion-immo`.

- [ ] **Step 3: Implement `mcpserver.Registry` and `cmd/gestion-immo/main.go` using `github.com/urfave/cli/v3` (`&cli.Command{Name: "gestion-immo", Commands: []*cli.Command{serveCmd, migrateCmd, tokenCmd}}`)**

Wire constructor injection across all 7 modules (`auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, `accounting`), register all HTTP routes on `http.NewServeMux()`, register all module MCP tools on `mcpserver.Registry`, and bind CLI flags to environment variables via `cli.EnvVars(...)`.

- [ ] **Step 4: Run `make test` and `make build` to verify everything compiles and passes**

Run: `make test && make build`
Expected: PASS and binary created at `bin/gestion-immo`.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/mcpserver cmd/gestion-immo
git commit -m "feat(cli,mcp): wire modular monolith with urfave/cli/v3 and Streamable HTTP MCP server"
```

---

### Task 10: SvelteKit SSR Frontend (`shadcn-svelte`) on Cloudflare Pages

**Files:**
- Create: `web/package.json`
- Create: `web/svelte.config.js`
- Create: `web/vite.config.ts`
- Create: `web/src/app.d.ts`
- Create: `web/src/lib/api/client.ts`
- Create: `web/src/lib/api/statement-state.ts`
- Create: `web/src/lib/api/statement-state.test.ts`
- Create: `web/src/routes/+layout.server.ts`
- Create: `web/src/routes/+layout.svelte`
- Create: `web/src/routes/login/+page.svelte`
- Create: `web/src/routes/signup/+page.svelte`
- Create: `web/src/routes/admin/users/+page.svelte`
- Create: `web/src/routes/+page.svelte`
- Create: `web/src/routes/entities/[id]/+page.svelte`
- Create: `web/src/routes/payments/negative-positions/+page.svelte`
- Create: `web/src/routes/inspections/[id]/+page.svelte`
- Create: `web/src/routes/accounting/[entityId]/statements/+page.svelte`

**Interfaces:**
- Consumes: Go REST API (`/api/v1/*`) via Cloudflare Tunnel URL (`PUBLIC_API_BASE_URL`)
- Produces:
  - `toggleStatementLine(disabledIds: ReadonlySet<string>, lineId: string): ReadonlySet<string>` in `web/src/lib/api/statement-state.ts`
  - SSR-authenticated back-office pages using `shadcn-svelte` (`Table`, `Card`, `Badge`, `Dialog`, `Switch`, `Tabs`, `Button`)

- [ ] **Step 1: Write failing Vitest unit test for immutable statement line toggle state in `web/src/lib/api/statement-state.test.ts`**

```typescript
// web/src/lib/api/statement-state.test.ts
import { describe, expect, it } from 'vitest';
import { toggleStatementLine, toDisabledArray } from './statement-state';

describe('statement-state line toggling', () => {
	it('immutably adds and removes disabled line IDs for dynamic statement recalculation', () => {
		const initial = new Set<string>();
		const afterDisable = toggleStatementLine(initial, 'rev_comm');
		expect(initial.size).toBe(0);
		expect(toDisabledArray(afterDisable)).toEqual(['rev_comm']);

		const afterReenable = toggleStatementLine(afterDisable, 'rev_comm');
		expect(toDisabledArray(afterReenable)).toEqual([]);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `make web-check`
Expected: FAIL with missing module `./statement-state`.

- [ ] **Step 3: Scaffold SvelteKit project in `web/` with `@sveltejs/adapter-cloudflare` and `shadcn-svelte`, implement `client.ts`, `statement-state.ts`, and the core back-office routes**

Implement:
- `web/src/routes/+page.svelte`: Portfolio overview across legal entities (`SARL INVESTISSIMO`, `SCI YOSSI`, etc.), quick cash rent entry modal, and negative positions alert banner.
- `web/src/routes/payments/negative-positions/+page.svelte`: Unpaid rent table showing exact unpaid months, cash/CSV payment buttons, and 1-click **Générer Dossier GLI** button.
- `web/src/routes/inspections/[id]/+page.svelte`: Room-by-room inspection template grid displaying photos and observations pushed by Antigravity MCP + PDF export button.
- `web/src/routes/accounting/[entityId]/statements/+page.svelte`: Interactive *Comptes Annuels* preview (Bilan Actif/Passif détaillé, Détail du Compte de Résultat, SIG) with a `shadcn-svelte` `Switch` on every line that calls `POST /api/v1/entities/{id}/statements/preview` to dynamically update subtotals, net result, and `%` variations before downloading the final PDF or FEC.

- [ ] **Step 4: Run frontend tests and typecheck via `Makefile`**

Run: `make web-check`
Expected: PASS with 0 errors.

- [ ] **Step 5: Commit**

```bash
git add web/
git commit -m "feat(web): add SvelteKit SSR frontend with shadcn-svelte, negative positions view, and interactive statement line toggling"
```
