# PostgreSQL 18 & Native RFC 9562 UUIDv7 Identity Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade PostgreSQL to `postgres:18-alpine` with a clean greenfield `UUIDv7` SQL schema (`001`–`006`), replace all `<prefix>_<hex>` entity IDs with RFC 9562 `UUIDv7` (`github.com/google/uuid`) across all 7 Go modules, extract 256-bit bearer token generation to `internal/platform/secret`, guarantee `4xx` LSP parity on malformed/missing UUIDs, and synchronize OpenAPI 3.1 + SvelteKit short-ID formatting.

**Architecture:** Entities mint monotonic RFC 9562 `UUIDv7` identifiers at birth in Go via `internal/platform/id.NewV7()` (wrapping `github.com/google/uuid`) while PostgreSQL 18 backs all 24 tables with native 16-byte `UUID PRIMARY KEY DEFAULT uuidv7()` columns and real nullable `UUID REFERENCES ... ON DELETE SET NULL` foreign keys. Opaque secrets (`secret.GenerateToken` for unprefixed 256-bit HTTP refresh tokens, `secret.GeneratePrefixedToken("mcp_live")` for MCP PATs) and human cash receipt suffixes (`payment`) are decoupled from `internal/platform/id` per SRP. Application services and PostgreSQL repositories guard `id.IsValidUUID` and translate `SQLSTATE 23503` via `postgres.IsForeignKeyViolation` into `domain.ErrValidation` (`422`) so neither `pgx/v5` `UUIDCodec` errors nor SQL FK errors ever leak as `500`.

**Tech Stack:** Go 1.27 (`github.com/google/uuid` v1.6.0+, `jackc/pgx/v5`, `danielgtaylor/huma/v2`), PostgreSQL 18 (`postgres:18-alpine`), `jackc/tern/v2`, SvelteKit 5 (`openapi-typescript`, Vitest).

**Spec:** `docs/superpowers/specs/2026-10-30-uuidv7-postgres18-id-architecture-design.md`

## Global Constraints

- **Strictly English in Code (`AGENTS.md` Rule 1):** All Go identifiers, SQL table/column/index/constraint names, OpenAPI schemas, and TypeScript symbols MUST be 100% English (eliminating legacy `"edl"` / `"edli"` prefixes).
- **Monetary Arithmetic (`AGENTS.md` Rule 2):** Preserve `money.AmountCents` and `money.BasisPoints` integer types untouched.
- **Explicit DI & External Migrations (`AGENTS.md` Rules 3 & 4):** Do not add new cross-module DI dependencies or embed `tern` migrations inside the Go binary.
- **Zero Dead Code & Mandatory Tooling (`AGENTS.md` Rule 5):** Delete `api/migrations/007_schema_constraints_and_indexes.sql`, `id.Generate`, and `isSafeScopeID`; run `npm --prefix web run format` immediately after any `web/` edit; do not touch `web/src/lib/components/ui/**`.
- **Non-UUID Text Identifiers Preserved:** `journal_lines.line_id`, `account_balance_lines.line_id`, `annual_account_reports.disabled_line_ids`, `account_code`, `receipt_number`, `siret`, `token_hash`, `r2_key`, and SvelteKit's `'ALL'` cookie sentinel MUST remain `TEXT` / `TEXT[]` / strings.

## Review Focus

1. **Non-canonical UUID forms accepted by `google/uuid.Validate` (`urn:uuid:...`, `{...}`, 32-char hyphenless hex):** `id.IsValidUUID` must return `false` and accept only canonical 36-char `8-4-4-4-12` UUIDs (tested in Task 1).
2. **Optional-ID REST requests sending `""` (`{"template_id": ""}` on `POST /api/v1/leases/{id}/gli-dossier` or `GET /api/v1/incidents` without filters):** must succeed (`201` with `template_id == ""` / `200`) without Huma `format:"uuid"` rejecting `""` as `422` or writing `"tpl_gli_default"` (tested in Task 5).
3. **Malformed non-UUID string (`"not-a-uuid"`, `"ent_1"`) passed directly to PostgreSQL repository `Find*ByID`, `List*By*`, or `ListLotsByPropertyIDs`:** must return `Err...NotFound` or empty slice/map without `pgx/v5` client-side `UUIDCodec` failure or SQL `22P02` (tested in Task 5).
4. **Valid `UUIDv7` referencing a non-existent cross-module parent row in PostgreSQL (`SQLSTATE 23503`) or `GenerateAnnualAccountsPDF` with an unknown entity UUID:** must map `23503` to `domain.ErrValidation` (`422`) and fail with `ErrLegalEntityNotFound` (`404`) before uploading orphaned PDFs to R2 (tested in Task 5).
5. **Two `UUIDv7` IDs minted within the same millisecond formatted via `formatShortId` in SvelteKit UI fallbacks:** must yield distinct trailing random hex labels rather than colliding on the shared timestamp prefix (tested in Task 6).

---

### Task 1: Platform Foundations (`internal/platform/id`, `internal/platform/secret`, `internal/platform/postgres`)

**Files:**
- Modify: `api/go.mod`, `api/go.sum`
- Modify: `api/internal/platform/id/id.go`
- Modify: `api/internal/platform/id/id_test.go`
- Create: `api/internal/platform/secret/token.go`
- Create: `api/internal/platform/secret/secret_test.go`
- Create: `api/internal/platform/postgres/uuid.go`
- Create: `api/internal/platform/postgres/uuid_test.go`

**Interfaces:**
- Consumes: `github.com/google/uuid` (`v1.6.0+`), `github.com/jackc/pgx/v5/pgconn`
- Produces:
  - `id.NewV7() (string, error)`
  - `id.MustNewV7() string`
  - `id.IsValidUUID(s string) bool`
  - *(Temporary compatibility until Task 5 removes the last caller: keep `id.Generate(prefix string) (string, error)` in `id.go` so `go test ./...` stays compilable across Tasks 1–4)*
  - `secret.GenerateToken() (string, error)` (32 random bytes → 64 lowercase hex chars, unprefixed)
  - `secret.GeneratePrefixedToken(prefix string) (string, error)` (`<trimmedPrefix>_<64-lowercase-hex>`)
  - `postgres.NullableUUID(s string) *string`
  - `postgres.DerefString(p *string) string`
  - `postgres.IsForeignKeyViolation(err error) bool` (`pgErr.Code == "23503"`)

- [ ] **Step 1: Add `github.com/google/uuid` to `api/go.mod`**

Run: `go -C api get github.com/google/uuid@latest`
Expected: `github.com/google/uuid v1.6.0` (or newer `v1.x`) added cleanly to `api/go.mod` and `api/go.sum`.

- [ ] **Step 2: Write failing unit tests for `id`, `secret`, and `postgres` helpers**

In `api/internal/platform/id/id_test.go`, test only our wrapper contract (`NewV7` returns a valid canonical string and `IsValidUUID` rejects non-canonical forms accepted by `uuid.Validate` alone):
```go
func TestIsValidUUID_StrictCanonical36Char(t *testing.T) {
	sample, err := id.NewV7()
	if err != nil {
		t.Fatalf("NewV7 failed: %v", err)
	}
	noHyphens := strings.ReplaceAll(sample, "-", "")
	cases := []struct {
		in   string
		want bool
	}{
		{sample, true},
		{strings.ToUpper(sample), true},
		{"", false},
		{"ent_0123456789abcdef0123456789abcdef", false},
		{"urn:uuid:" + sample, false},
		{"{" + sample + "}", false},
		{noHyphens, false},
		{" " + sample, false},
	}
	for _, tc := range cases {
		if got := id.IsValidUUID(tc.in); got != tc.want {
			t.Errorf("IsValidUUID(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
```

In `api/internal/platform/secret/secret_test.go`, test:
- `GenerateToken()` returns a 64-char lowercase hex string (`32` bytes), contains no `_`, and produces distinct values across consecutive calls.
- `GeneratePrefixedToken("mcp_live")` returns `"mcp_live_" + <64-hex>` (`73` chars total), and `GeneratePrefixedToken("   ")` returns an error.
- `secret.New("raw-secret").String()` returns `secret.RedactedPlaceholder`.

In `api/internal/platform/postgres/uuid_test.go`, test:
- `NullableUUID("")` and `NullableUUID("   ")` return `nil`; `NullableUUID(" 0192dce0-0000-7000-8000-000000000001 ")` returns pointer to trimmed string.
- `DerefString(nil)` returns `""`; `DerefString(&s)` returns `s`.
- `IsForeignKeyViolation(fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "23503"}))` returns `true`, while `Code: "23505"` and `errors.New("other")` return `false`.

- [ ] **Step 3: Run platform tests to verify RED**

Run: `go -C api test ./internal/platform/id/... ./internal/platform/secret/... ./internal/platform/postgres/...`
Expected: FAIL with undefined `id.NewV7`, `id.MustNewV7`, `id.IsValidUUID`, `secret.GenerateToken`, `secret.GeneratePrefixedToken`, `postgres.NullableUUID`, `postgres.DerefString`, `postgres.IsForeignKeyViolation`.

- [ ] **Step 4: Implement `id`, `secret`, and `postgres` helpers**

- `api/internal/platform/id/id.go`: implement `NewV7() (string, error)` via `uuid.NewV7()`, `MustNewV7() string`, and `IsValidUUID(s string) bool` checking `len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' && uuid.Validate(s) == nil`. Keep `Generate(prefix string)` marked deprecated until Task 5 deletes it.
- `api/internal/platform/secret/token.go`: implement `GenerateToken() (string, error)` using `bytes := make([]byte, 32)`, `rand.Read(bytes)`, `hex.EncodeToString(bytes)`, and `GeneratePrefixedToken(prefix string) (string, error)` delegating to `GenerateToken()`.
- `api/internal/platform/postgres/uuid.go`: implement `NullableUUID(s string) *string`, `DerefString(p *string) string`, and `IsForeignKeyViolation(err error) bool`.

- [ ] **Step 5: Run tests to verify GREEN**

Run: `make -C api test && make -C api lint`
Expected: PASS across all Go packages.

- [ ] **Step 6: Commit**

```bash
git add api/go.mod api/go.sum api/internal/platform/id/ api/internal/platform/secret/ api/internal/platform/postgres/uuid.go api/internal/platform/postgres/uuid_test.go
git commit -m "feat(platform): add UUIDv7 helpers, 256-bit secret tokens, and Postgres UUID utilities"
```

---

### Task 2: PostgreSQL 18 Docker Upgrade & Greenfield `tern` SQL Schema (`001`–`006`)

**Files:**
- Modify: `api/docker-compose.yml`
- Modify: `README.md`
- Modify: `api/migrations/001_auth_and_users.sql`
- Modify: `api/migrations/002_portfolio_hierarchy.sql`
- Modify: `api/migrations/003_leases_and_tenants.sql`
- Modify: `api/migrations/004_payments_and_cash_journal.sql`
- Modify: `api/migrations/005_templates_inspections_incidents.sql`
- Modify: `api/migrations/006_accounting_and_annual_accounts.sql`
- Delete: `api/migrations/007_schema_constraints_and_indexes.sql`
- Create: `api/migrations/migrations_schema_test.go` (or `api/internal/platform/postgres/migrations_test.go`)

**Interfaces:**
- Consumes: Spec §4.1 and §4.3 24-table UUID matrix
- Produces: Greenfield `001`–`006` PostgreSQL 18 `tern` migrations with `UUID PRIMARY KEY DEFAULT uuidv7()` across all 24 tables, nullable `UUID REFERENCES ... ON DELETE SET NULL` FKs, and `postgres:18-alpine` Docker configuration

- [ ] **Step 1: Write a failing schema-contract test in `api/internal/platform/postgres/migrations_test.go`**

Write `TestMigrations_EnforceUUIDv7AndNoLegacyTextIDs(t *testing.T)` reading `../../../migrations/*.sql` and asserting:
- Exactly 6 migration files exist (`001_auth_and_users.sql` through `006_accounting_and_annual_accounts.sql`; `007_schema_constraints_and_indexes.sql` does not exist).
- `pcg_accounts` does not appear in any migration file.
- `id TEXT PRIMARY KEY` does not appear in any migration file; `id UUID PRIMARY KEY DEFAULT uuidv7()` appears 24 times across the 6 files (once per table).
- `stored_files`, `gli_dossiers`, and `incidents` define nullable `UUID REFERENCES` foreign keys (`legal_entity_id UUID REFERENCES legal_entities(id) ON DELETE SET NULL`, `lease_id UUID REFERENCES leases(id) ON DELETE SET NULL`, `template_id UUID REFERENCES document_templates(id) ON DELETE SET NULL`, `property_id UUID REFERENCES properties(id) ON DELETE SET NULL`, `lot_id UUID REFERENCES lots(id) ON DELETE SET NULL`) with zero `legal_entity_id TEXT`, `lease_id TEXT`, `template_id TEXT`, `property_id TEXT`, or `lot_id TEXT`.
- `line_id TEXT` is preserved in `journal_lines` and `account_balance_lines`, and `disabled_line_ids TEXT[]` is preserved in `annual_account_reports`.

- [ ] **Step 2: Run the schema test to verify RED**

Run: `go -C api test -v ./internal/platform/postgres -run TestMigrations_EnforceUUIDv7AndNoLegacyTextIDs`
Expected: FAIL (`007_schema_constraints_and_indexes.sql` still exists, `id TEXT PRIMARY KEY` still present).

- [ ] **Step 3: Upgrade `api/docker-compose.yml`, `README.md`, and rewrite `api/migrations/001..006`**

1. In `api/docker-compose.yml`:
   - Change `services.postgres.image` and `services.pg-backup.image` to `postgres:18-alpine`.
   - Change `services.postgres.volumes` mount from `pg_data:/var/lib/postgresql/data` to `pg_data:/var/lib/postgresql`.
2. In `README.md`: update `PostgreSQL 16` → `PostgreSQL 18` (mentioning native `UUIDv7` primary keys and `docker compose -f api/docker-compose.yml down -v` when resetting a legacy PG 16 local volume).
3. Rewrite `api/migrations/001_auth_and_users.sql` .. `006_accounting_and_annual_accounts.sql` per Spec §4.3:
   - Every table PK: `id UUID PRIMARY KEY DEFAULT uuidv7()`.
   - Inline former `007` constraints & indexes directly into:
     - `002_portfolio_hierarchy.sql`: `property_type TEXT NOT NULL CHECK (property_type IN ('APARTMENT', 'HOUSE', 'COMMERCIAL_PREMISES', 'BARE_LAND', 'PARKING', 'CELLAR', 'BUILDING'))` on `properties`; `lot_type TEXT NOT NULL DEFAULT 'APARTMENT' CHECK (lot_type IN ('APARTMENT', 'HOUSE', 'COMMERCIAL_PREMISES', 'BARE_LAND', 'PARKING', 'CELLAR', 'BUILDING'))` on `lots`.
     - `003_leases_and_tenants.sql`: `idx_rent_calls_tenant_id ON rent_calls(tenant_id)` and `idx_rent_calls_lease_unpaid_order ON rent_calls(lease_id, due_date ASC, period_month ASC) WHERE remaining_due_cents > 0 AND status <> 'PAID'`.
     - `004_payments_and_cash_journal.sql`: `idx_cash_journal_payment_id ON cash_journal_entries(payment_id) WHERE payment_id IS NOT NULL`.
     - `005_templates_inspections_incidents.sql`: `inspections.lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE`, partial index `idx_inspections_entry_inspection_id ON inspections(entry_inspection_id) WHERE entry_inspection_id IS NOT NULL`, nullable `UUID REFERENCES ... ON DELETE SET NULL` on `stored_files.{legal_entity_id, lease_id}`, `gli_dossiers.template_id` (+ `idx_gli_dossiers_template_id`), and `incidents.{property_id, lot_id, lease_id}` (+ `idx_incidents_lot_id`).
     - `001_auth_and_users.sql`: partial indexes on `users(approved_by) WHERE approved_by IS NOT NULL` and `refresh_tokens(replaced_by_id) WHERE replaced_by_id IS NOT NULL`; `family_id UUID NOT NULL`.
     - `006_accounting_and_annual_accounts.sql`: omit `pcg_accounts`; keep `line_id TEXT` and `disabled_line_ids TEXT[]`.
4. Delete `api/migrations/007_schema_constraints_and_indexes.sql`.

- [ ] **Step 4: Run schema tests and Go suite to verify GREEN**

Run: `make -C api test && make -C api lint`
Expected: PASS (`TestMigrations_EnforceUUIDv7AndNoLegacyTextIDs` and all package tests pass).

- [ ] **Step 5: Commit**

```bash
git add api/docker-compose.yml README.md api/migrations/ api/internal/platform/postgres/migrations_test.go
git commit -m "feat(db): upgrade to PostgreSQL 18 and rewrite greenfield UUIDv7 tern migrations"
```

---

### Task 3: Migrate `auth` & `portfolio` Modules (`UUIDv7`, 256-Bit `secret` Tokens, LSP Guards & `format:"uuid"`)

**Files:**
- Modify: `api/internal/modules/auth/application/service.go`
- Modify: `api/internal/modules/auth/infrastructure/postgres/user_repo.go`
- Modify: `api/internal/modules/auth/infrastructure/postgres/refresh_token_repo.go`
- Modify: `api/internal/modules/auth/infrastructure/postgres/mcp_token_repo.go`
- Modify: `api/internal/modules/auth/interfaces/http/handler.go`
- Modify: `api/internal/modules/auth/application/service_test.go`
- Modify: `api/internal/modules/auth/infrastructure/security_test.go`
- Modify: `api/internal/modules/portfolio/application/service.go`
- Modify: `api/internal/modules/portfolio/infrastructure/postgres/legal_entity_repo.go`
- Modify: `api/internal/modules/portfolio/infrastructure/postgres/shareholder_repo.go`
- Modify: `api/internal/modules/portfolio/infrastructure/postgres/property_repo.go`
- Modify: `api/internal/modules/portfolio/infrastructure/postgres/lot_repo.go`
- Modify: `api/internal/modules/portfolio/interfaces/http/handler.go`
- Modify: `api/internal/modules/portfolio/application/service_test.go`

**Interfaces:**
- Consumes: `id.NewV7`, `id.MustNewV7`, `id.IsValidUUID`, `secret.GenerateToken`, `secret.GeneratePrefixedToken`, `postgres.IsForeignKeyViolation`
- Produces: `auth` & `portfolio` services minting `UUIDv7` entity IDs, unprefixed 64-hex refresh tokens, `mcp_live_<64-hex>` MCP tokens, `id.IsValidUUID` validation, `postgres` repository LSP guards, and `format:"uuid"` OpenAPI tags

- [ ] **Step 1: Write failing tests in `auth/application/service_test.go` and `portfolio/application/service_test.go`**

Add assertions verifying:
- `auth`: created user `ID`, refresh token record/family IDs, and MCP token `ID` satisfy `id.IsValidUUID(...)` (`got[14] == '7'`); raw refresh token from `Login` is a 64-character hex string with **no** `rt_live_` prefix (`!strings.Contains(pair.RefreshToken, "_") && len(pair.RefreshToken) == 64`); raw MCP token from `CreateMCPToken` starts with `"mcp_live_"` and has total length `73`; `ApproveUser`/`RejectUser`/`CreateMCPToken`/`RevokeMCPToken` reject malformed non-UUID IDs (`ErrValidation` or `ErrNotFound` consistently).
- `portfolio`: created `LegalEntityDTO.ID`, `ShareholderDTO.ID`, `PropertyDTO.ID`, and `LotDTO.ID` satisfy `id.IsValidUUID(...)`; `UpsertShareholder`, `CreateProperty`, `ListProperties`, and `CreateLot` reject malformed `LegalEntityID` / `PropertyID` (`"not-a-uuid"`) with `domain.ErrValidation`, while `GetEntityPortfolioSummary(ctx, "not-a-uuid")` returns `domain.ErrLegalEntityNotFound`. Update any legacy `"ent_..."` fixtures in `portfolio/application/service_test.go` to `id.MustNewV7()`.

- [ ] **Step 2: Run `auth` and `portfolio` tests to verify RED**

Run: `go -C api test ./internal/modules/auth/... ./internal/modules/portfolio/...`
Expected: FAIL (IDs still `"usr_..."`, `"ent_..."`; refresh token still `"rt_live_..."`).

- [ ] **Step 3: Implement `UUIDv7`, `secret` token separation, LSP guards, and Huma `format:"uuid"` in `auth` & `portfolio`**

1. **`auth`:**
   - In `auth/application/service.go`: replace all 5 entity/family `id.Generate(...)` calls (`usr`, `mcptok`, `rtfam`, `rt`) with `id.NewV7()`; replace `id.Generate("mcp_live")` with `secret.GeneratePrefixedToken("mcp_live")`; replace `id.Generate("rt_live")` with `secret.GenerateToken()`. Validate `id.IsValidUUID` on required `userID` / `approverID` / `tokenID` inputs (`domain.ErrValidation`).
   - In `auth/infrastructure/postgres/{user_repo,refresh_token_repo,mcp_token_repo}.go`: guard `FindUserByID`, `UpdateUser`, `RotateRefreshToken`, `RevokeTokenFamily`, `RevokeUserRefreshTokens`, `ListMCPTokensByUser`, `RevokeMCPToken` with `!id.IsValidUUID(...)` returning `domain.ErrUserNotFound` / `domain.ErrInvalidToken` / `[]domain.MCPApiToken{}` / `domain.ErrTokenNotFound` without hitting Postgres; map `postgres.IsForeignKeyViolation(err)` on writes to `domain.ErrValidation`.
   - In `auth/interfaces/http/handler.go` (and `application` DTOs): add `format:"uuid"` on `path:"id"`, `UserDTO.ID`, `UserDTO.ApprovedBy`, and `MCPTokenDTO.ID` (never on raw token strings!). Update `auth/infrastructure/security_test.go` sample refresh token string to an unprefixed hex token.
2. **`portfolio`:**
   - In `portfolio/application/service.go`: replace all 4 `id.Generate(...)` calls (`ent`, `sh`, `prop`, `lot`) with `id.NewV7()`; validate `id.IsValidUUID` on `LegalEntityID` (`UpsertShareholder`, `CreateProperty`, `ListProperties`) and `PropertyID` (`CreateLot`) returning `domain.ErrValidation`.
   - In `portfolio/infrastructure/postgres/{legal_entity_repo,shareholder_repo,property_repo,lot_repo}.go`: guard `Find*ByID` (`!id.IsValidUUID` → `Err...NotFound`), `List*ByEntity` / `ListLotsByProperty` (`!id.IsValidUUID` → empty slice), and `ListLotsByPropertyIDs` (filter valid UUIDs, return empty map if none, use `WHERE property_id = ANY($1::uuid[])`); map `postgres.IsForeignKeyViolation(err)` on writes to `domain.ErrValidation`.
   - In `portfolio/interfaces/http/handler.go` and `portfolio/application` DTOs: add `format:"uuid"` on `path:"id"` and all `ID` / `LegalEntityID` / `PropertyID` DTO fields.

- [ ] **Step 4: Run tests to verify GREEN**

Run: `go -C api test ./internal/modules/auth/... ./internal/modules/portfolio/... && make -C api lint`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api/internal/modules/auth/ api/internal/modules/portfolio/
git commit -m "feat(auth,portfolio): migrate entity IDs to UUIDv7 and decouple 256-bit auth tokens"
```

---

### Task 4: Migrate `lease` & `payment` Modules (`UUIDv7`, Cash Receipt Suffix, LSP Guards & `format:"uuid"`)

**Files:**
- Modify: `api/internal/modules/lease/application/service.go`
- Modify: `api/internal/modules/lease/infrastructure/postgres/tenant_repo.go`
- Modify: `api/internal/modules/lease/infrastructure/postgres/lease_repo.go`
- Modify: `api/internal/modules/lease/infrastructure/postgres/rent_call_repo.go`
- Modify: `api/internal/modules/lease/interfaces/http/handler.go`
- Modify: `api/internal/modules/lease/application/service_test.go`
- Modify: `api/internal/modules/payment/application/service.go`
- Modify: `api/internal/modules/payment/infrastructure/postgres/payment_repo.go`
- Modify: `api/internal/modules/payment/infrastructure/postgres/cash_journal_repo.go`
- Modify: `api/internal/modules/payment/interfaces/http/handler.go`
- Modify: `api/internal/modules/payment/application/service_test.go`
- Modify: `api/cmd/main_test.go`

**Interfaces:**
- Consumes: `id.NewV7`, `id.MustNewV7`, `id.IsValidUUID`, `postgres.IsForeignKeyViolation`
- Produces: `lease` & `payment` services minting `UUIDv7` IDs, local `generateCashReceiptSuffix()` (`CAISSE-YYYYMMDD-<6-UPPER-HEX>`), `id.IsValidUUID` validation, `postgres` repository LSP guards, and `format:"uuid"` OpenAPI tags

- [ ] **Step 1: Update fixtures and add failing UUIDv7 / validation tests in `lease`, `payment`, and `cmd/main_test.go`**

- Replace legacy prefixed fixtures (`"ent_1"`, `"lot_1"`, `"lse_1"`, `"ent_main_auto"`, etc.) in `lease/application/service_test.go`, `payment/application/service_test.go`, and `api/cmd/main_test.go` with valid UUIDv7 values (`id.MustNewV7()` or deterministic RFC 9562 UUIDv7 constants such as `"0192dce0-0000-7000-8000-000000000001"`).
- Add assertions in `lease/application/service_test.go` verifying created `TenantDTO.ID`, `LeaseDTO.ID`, and `RentCallDTO.ID` satisfy `id.IsValidUUID`, and `CreateLease` / `GenerateMonthlyRentCalls` / `ListLeases` (with non-empty invalid `entityID`) return `domain.ErrValidation` on malformed UUIDs.
- Add assertions in `payment/application/service_test.go` verifying `PaymentReceiptDTO.ID`, `RentAllocationDTO.ID`, and `CashJournalEntryDTO.ID` satisfy `id.IsValidUUID`, auto-generated cash `ReceiptNumber` matches `^CAISSE-\d{8}-[0-9A-F]{6}$`, and `RecordCashPayment` / `RecordBankPayment` / `RecordCashDeposit` / `ImportBankTransactionsCSV` reject malformed UUIDs with `domain.ErrValidation`.

- [ ] **Step 2: Run `lease`, `payment`, and `cmd` tests to verify RED**

Run: `go -C api test ./internal/modules/lease/... ./internal/modules/payment/... ./cmd/...`
Expected: FAIL (generated IDs still `"ten_..."`, `"lse_..."`, `"pay_..."`, cash receipt suffix lowercase from `id.Generate("rcu")`, malformed UUIDs not rejected).

- [ ] **Step 3: Implement `UUIDv7`, `generateCashReceiptSuffix`, LSP guards, and `format:"uuid"` in `lease` & `payment`**

1. **`lease`:**
   - In `lease/application/service.go`: replace `id.Generate("ten")`, `id.Generate("lse")`, `id.Generate("rc")` with `id.NewV7()`; validate `id.IsValidUUID` on required `LegalEntityID`, `LotID`, `TenantID` (`CreateLease`, `GenerateMonthlyRentCalls`, `ListEntityRentCalls`) and optional `entityID` (`ListLeases`: if `trimmed != ""` && `!id.IsValidUUID(trimmed)` → `domain.ErrValidation`); on `ArchiveTenant` / `GetTenantByID` / `GetLeaseByID`, if `!id.IsValidUUID(trimmed)` return `ErrTenantNotFound` / `ErrLeaseNotFound` (or `ErrValidation` if empty, preserving existing empty-ID checks).
   - In `lease/infrastructure/postgres/{tenant_repo,lease_repo,rent_call_repo}.go`: guard `Find*ByID` / `ArchiveTenant` (`!id.IsValidUUID` → `Err...NotFound`), `List*ByEntity` / `List*ByLease` (`!id.IsValidUUID` → empty slice), and map `postgres.IsForeignKeyViolation(err)` on writes/batches to `domain.ErrValidation`.
   - In `lease/interfaces/http/handler.go` & DTOs: add `format:"uuid"` on `path:"id"`, required request UUID fields (`CreateLeaseCmd.LegalEntityID`, `LotID`, `TenantID`), and response DTO UUID fields (`TenantDTO.ID`, `LeaseDTO.{ID,LegalEntityID,LotID,TenantID}`, `RentCallDTO.{ID,LeaseID,LegalEntityID,TenantID}`). Omit `format:"uuid"` on optional `query:"entity_id"`.
2. **`payment`:**
   - In `payment/application/service.go`: replace `id.Generate("cje")` (2 sites), `id.Generate("pay")`, `id.Generate("alc")` with `id.NewV7()`; replace `id.Generate("rcu")` with local `generateCashReceiptSuffix() (string, error)` (`bytes := make([]byte, 3)` + `strings.ToUpper(hex.EncodeToString(bytes))`); validate `id.IsValidUUID` on `LeaseID` and `LegalEntityID` across payment, cash deposit, CSV import, negative positions, and cash journal methods (`domain.ErrValidation`).
   - In `payment/infrastructure/postgres/{payment_repo,cash_journal_repo}.go`: guard `Find*ByID` / `List*By*` (`!id.IsValidUUID` → `ErrPaymentNotFound` / empty slice), use `WHERE payment_id = ANY($1::uuid[])` in `loadAllocationsByPaymentIDs`, and map `postgres.IsForeignKeyViolation(err)` on writes to `domain.ErrValidation`.
   - In `payment/interfaces/http/handler.go` & DTOs: add `format:"uuid"` on `path:"id"`, required request UUIDs (`LeaseID`, `LegalEntityID`, `EntityID` in `ImportCSVBody`), and all response DTO UUIDs (`PaymentReceiptDTO`, `RentAllocationDTO`, `CashJournalEntryDTO` including `PaymentID *string`, `NegativePositionDTO`).

- [ ] **Step 4: Run tests to verify GREEN**

Run: `go -C api test ./internal/modules/lease/... ./internal/modules/payment/... ./cmd/... && make -C api lint`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api/internal/modules/lease/ api/internal/modules/payment/ api/cmd/main_test.go
git commit -m "feat(lease,payment): migrate entity IDs to UUIDv7 and decouple cash receipt suffixes"
```

---

### Task 5: Migrate `document`, `inspection`, & `accounting` Modules, Delete `id.Generate`, & Verify PostgreSQL 18 Integration

**Files:**
- Modify: `api/internal/platform/id/id.go`
- Modify: `api/internal/platform/id/id_test.go`
- Modify: `api/internal/modules/document/application/service.go`
- Modify: `api/internal/modules/document/infrastructure/postgres/document_template_repo.go`
- Modify: `api/internal/modules/document/infrastructure/postgres/stored_file_repo.go`
- Modify: `api/internal/modules/document/infrastructure/postgres/gli_dossier_repo.go`
- Modify: `api/internal/modules/document/interfaces/http/handler.go`
- Modify: `api/internal/modules/document/application/service_test.go`
- Modify: `api/internal/modules/inspection/application/service.go`
- Modify: `api/internal/modules/inspection/infrastructure/postgres/inspection_template_repo.go`
- Modify: `api/internal/modules/inspection/infrastructure/postgres/inspection_repo.go`
- Modify: `api/internal/modules/inspection/infrastructure/postgres/incident_repo.go`
- Modify: `api/internal/modules/inspection/interfaces/http/handler.go`
- Modify: `api/internal/modules/inspection/application/service_test.go`
- Modify: `api/internal/modules/accounting/application/service.go`
- Modify: `api/internal/modules/accounting/infrastructure/memory/account_balance_repo.go`
- Modify: `api/internal/modules/accounting/infrastructure/postgres/account_balance_repo.go`
- Modify: `api/internal/modules/accounting/infrastructure/postgres/annual_account_report_repo.go`
- Modify: `api/internal/modules/accounting/infrastructure/postgres/journal_entry_repo.go`
- Modify: `api/internal/modules/accounting/interfaces/http/handler.go`
- Modify: `api/internal/modules/accounting/application/service_test.go`
- Modify: `api/internal/platform/postgres/integration_test.go`

**Interfaces:**
- Consumes: `id.NewV7`, `id.MustNewV7`, `id.IsValidUUID`, `postgres.NullableUUID`, `postgres.DerefString`, `postgres.IsForeignKeyViolation`
- Produces: Zero remaining `id.Generate` references anywhere in `api/`; full nullable-UUID & `23503` / malformed-UUID LSP coverage across `document`, `inspection`, `accounting`, and `postgres/integration_test.go`

- [ ] **Step 1: Update fixtures and write failing tests in `document`, `inspection`, `accounting`, and `postgres/integration_test.go`**

- **`document/application/service_test.go`:**
  - Update legacy prefixed IDs (`"usr_admin"`, etc.) to valid UUIDv7 values.
  - Assert `POST /api/v1/leases/{id}/gli-dossier` with `{"template_id": ""}` returns `201 Created` with `TemplateID == ""` (not `"tpl_gli_default"`), while passing a valid-syntax non-existent `template_id` UUID returns `404 Not Found` (`domain.ErrTemplateNotFound`).
  - Assert `AttachFileToR2` rejects any non-UUID `LeaseID` or `LegalEntityID` (including `"../secret"`, `"a/b"`, and `"lease_1"`) with `domain.ErrValidation` before uploading to R2.
- **`inspection/application/service_test.go`:**
  - Update legacy `"lease_..."`, `"prop_..."`, `"lot_..."` fixtures to `id.MustNewV7()`.
  - Assert created `InspectionTemplateDTO.ID`, `InspectionDTO.ID`, `InspectionItemDTO.ID`, and `IncidentDTO.ID` satisfy `id.IsValidUUID` (zero `"edl_"` / `"edli_"` prefixes).
  - Assert `POST /api/v1/incidents` with omitted `lot_id` / `lease_id` (`""`) and `GET /api/v1/incidents` without query filters succeed (`201` / `200`), while non-empty malformed `property_id` or `lease_id` returns `422`.
- **`accounting/application/service_test.go`:**
  - Update legacy `"ent_..."` fixtures to `id.MustNewV7()`.
  - Assert `AccountBalanceLine.ID`, `JournalEntryDTO.ID`, `JournalLineDTO.ID`, and `AnnualAccountReport` IDs are valid `UUIDv7` values while `LineID` (`"rev_locaux"`) and `DisabledLineIDs` (`["chg_loc"]`) remain domain slugs.
  - Assert `GenerateAnnualAccountsPDF` when `portfolioSvc != nil` and `entityID` is an unknown valid `UUIDv7` returns `portfoliodomain.ErrLegalEntityNotFound` (HTTP `404` via `accounting/interfaces/http/handler.go`) and uploads **zero** objects to R2.
- **`internal/platform/postgres/integration_test.go`:**
  - Convert all 24-table test fixtures to `id.MustNewV7()`.
  - Add sub-assertions for: (a) nullable `UUID` round-trip (`""` ↔ `SQL NULL`) on `stored_files` (with empty `lease_id`), `gli_dossiers` (with empty `template_id`), and `incidents` (with empty `lot_id`/`lease_id` + `ListIncidents(ctx, "", "")` and `ListIncidents(ctx, prop.ID, "")`); (b) malformed non-UUID inputs (`"not-a-uuid"`) to `FindUserByID`, `FindLegalEntityByID`, `ListLotsByPropertyIDs`, and `ListIncidents` returning `Err...NotFound` / empty results instead of `pgx` `UUIDCodec` errors; (c) valid-UUID foreign-key violation (`SaveInspection` with `id.MustNewV7()` unknown `LeaseID`) returning `inspectiondomain.ErrValidation`.

- [ ] **Step 2: Run tests to verify RED**

Run: `go -C api test ./internal/modules/document/... ./internal/modules/inspection/... ./internal/modules/accounting/...`
Expected: FAIL on `"tpl_gli_default"`, `"edl_"`/`"edli_"` prefixes, and missing pre-upload entity check.

- [ ] **Step 3: Implement `document`, `inspection`, & `accounting` migrations and delete `id.Generate`**

1. **`document`:**
   - `document/application/service.go`: replace `id.Generate` (`tpl`, `fil` ×2, `gli`) with `id.NewV7()`; replace `isSafeScopeID` with `id.IsValidUUID` and delete `isSafeScopeID`; validate `id.IsValidUUID` on `GenerateFromTemplate`, `AssembleGLIDossier`, `ListStoredFilesByLease`, and non-empty `SearchDocuments` IDs; in `resolveGLIDocxTemplate`, when `trimmed != ""`, validate `id.IsValidUUID(trimmed)` (`domain.ErrValidation`), propagate `domain.ErrTemplateNotFound` when missing, and when `trimmed == ""` return `data, "", nil` (eliminating `"tpl_gli_default"`).
   - `document/infrastructure/postgres/{document_template_repo,stored_file_repo,gli_dossier_repo}.go`: use `postgres.NullableUUID` on write and `*string` + `postgres.DerefString` on scan for `stored_files.{legal_entity_id, lease_id}` and `gli_dossiers.template_id`; guard `!id.IsValidUUID` on `Find*ByID` / `List*By*`; map `postgres.IsForeignKeyViolation` to `domain.ErrValidation`.
   - `document/interfaces/http/handler.go` & DTOs: add `format:"uuid"` ONLY on `path:"id"`, `GenerateDocCmd.TemplateID`, `AssembleGLIDossierBody.LeaseID`, `DocumentTemplateDTO.ID`, `StoredFileDTO.ID`, and `GLIDossierArchiveDTO.{ID,LeaseID,LegalEntityID}` (omit on optional plain-`string` `LegalEntityID`, `LeaseID`, `TemplateID`).
2. **`inspection`:**
   - `inspection/application/service.go`: replace `id.Generate` (`itpl`, `edl`, `edli`, `inc`) with `id.NewV7()`; validate `id.IsValidUUID` on required IDs (`CreateInspection`, `UpsertInspectionItems`, `GenerateInspectionPDF`, `UpdateIncident`) and non-empty optional IDs (`EntryInspectionID`, `CreateIncidentCmd.{PropertyID,LotID,LeaseID}`, `ListInspectionsByLease`, `ListIncidents`).
   - `inspection/infrastructure/postgres/{inspection_template_repo,inspection_repo,incident_repo}.go`: guard `!id.IsValidUUID`; use `WHERE inspection_id = ANY($1::uuid[])` in `inspection_repo.go`; in `incident_repo.go`, bind `PropertyID`, `LotID`, `LeaseID` via `postgres.NullableUUID`, scan via `*string` + `postgres.DerefString`, and rewrite `ListIncidents` filter to `WHERE ($1::uuid IS NULL OR property_id = $1) AND ($2::uuid IS NULL OR lease_id = $2)`; map `postgres.IsForeignKeyViolation` to `domain.ErrValidation`.
   - `inspection/interfaces/http/handler.go` & DTOs: add `format:"uuid"` ONLY on `path:"id"`, `CreateInspectionCmd.{LeaseID,TemplateID}`, `InspectionTemplateDTO.ID`, `InspectionDTO.{ID,LeaseID,TemplateID,EntryInspectionID}`, `InspectionItemDTO.{ID,InspectionID}`, `InspectionReportDTO.{InspectionID,LeaseID,EntryInspectionID}`, and `IncidentDTO.ID` (omit on optional plain-`string` `PropertyID`, `LotID`, `LeaseID`, `EntryInspectionID`, and query filters).
3. **`accounting`:**
   - `accounting/application/service.go`: replace `id.Generate` (`je`, `jl`, `aar`) with `id.NewV7()`; validate `id.IsValidUUID` on `LegalEntityID` across `RecordJournalEntry`, `RecordExpenseInvoice`, `ListJournalEntries`, `UpsertAccountBalanceLines`, `GetTrialBalance`, `PreviewFinancialStatements`, `GenerateAnnualAccountsPDF`, and `ExportEntityFEC` (do **NOT** validate `LineID` or `DisabledLineIDs` as UUIDs — they are PCG/CERFA slugs!); in `GenerateAnnualAccountsPDF`, when `s.portfolioSvc != nil`, call `s.portfolioSvc.GetEntityPortfolioSummary(ctx, trimmedEntity)` and return any error **before** rendering/uploading the PDF to R2.
   - `accounting/interfaces/http/handler.go`: add `errors.Is(err, portfoliodomain.ErrLegalEntityNotFound)` to the `http.StatusNotFound` branch of `mapDomainError`; add `format:"uuid"` on `path:"id"` and entity/entry/line primary/foreign `UUID` DTO fields (never on `line_id` or `disabled_line_ids`).
   - `accounting/infrastructure/postgres/{account_balance_repo,annual_account_report_repo,journal_entry_repo}.go` & `accounting/infrastructure/memory/account_balance_repo.go`: when `line.ID == ""`, assign `id.NewV7()` (preserving existing `ID` on update in `memory` and via `ON CONFLICT ... DO UPDATE` in `postgres`); guard `!id.IsValidUUID`; use `WHERE journal_entry_id = ANY($1::uuid[])` in `journal_entry_repo.go`; map `postgres.IsForeignKeyViolation` to `domain.ErrValidation`.
4. **Delete `id.Generate`:** remove `Generate(prefix string)` from `api/internal/platform/id/id.go` and remove its legacy test from `api/internal/platform/id/id_test.go`.

- [ ] **Step 4: Run full Go unit, race, lint, build, and PostgreSQL 18 integration verification**

1. Run: `grep -rn "id\.Generate" api/ || true`
   Expected: zero matches.
2. Run: `make -C api test && make -C api test-race && make -C api lint && make -C api build`
   Expected: PASS across all Go packages and binary build.
3. If Docker daemon is available locally, verify PostgreSQL 18 + `tern` round-trip and `integration_test.go`:
   - Reset/start PG 18 (`docker compose -f api/docker-compose.yml up -d postgres`), run `make -C api migrate-up`, `make -C api migrate-down`, `make -C api migrate-up`, and run `TEST_DATABASE_URL="postgres://..." go -C api test -v ./internal/platform/postgres -run TestPostgresIntegration_TransactionsAndRepositories`.

- [ ] **Step 5: Commit**

```bash
git add api/internal/platform/id/ api/internal/platform/postgres/integration_test.go api/internal/modules/document/ api/internal/modules/inspection/ api/internal/modules/accounting/
git commit -m "feat(api): complete UUIDv7 migration across document, inspection, accounting and delete id.Generate"
```

---

### Task 6: OpenAPI 3.1 Regeneration & SvelteKit `formatShortId` Frontend Migration

**Files:**
- Modify: `web/src/lib/core/api/openapi.json`
- Modify: `web/src/lib/core/api/openapi.d.ts`
- Create: `web/src/lib/core/formatters/id.ts`
- Create: `web/src/lib/core/formatters/id.test.ts`
- Modify: `web/src/routes/(app)/inspections/(components)/CreateInspectionDialog.svelte`
- Modify: `web/src/routes/(app)/inspections/(components)/InspectionsListCard.svelte`
- Modify: `web/src/routes/(app)/inspections/incidents/(components)/IncidentsTableCard.svelte`
- Modify: `web/src/routes/(app)/inspections/incidents/(components)/CreateIncidentDialog.svelte`
- Modify: `web/src/routes/(app)/rents/(components)/rent-rows.ts`
- Modify: `web/src/routes/(app)/leases/(components)/LeasesTableCard.svelte`
- Modify: `web/src/routes/(app)/rents/(components)/rent-rows.test.ts`
- Modify: `web/src/lib/modules/cockpit/cockpit-builder.test.ts`
- Modify: `web/src/lib/core/workspace/onboarding-gate.test.ts`
- Modify: `web/src/lib/server/backend-api.test.ts`

**Interfaces:**
- Consumes: Exported OpenAPI 3.1 JSON (`make -C api openapi`)
- Produces:
  - Regenerated `web/src/lib/core/api/openapi.json` and `web/src/lib/core/api/openapi.d.ts` with `@format uuid` annotations on required UUID fields
  - `formatShortId(id: string, length?: number): string` in `web/src/lib/core/formatters/id.ts`

- [ ] **Step 1: Regenerate `openapi.json` and `openapi.d.ts`**

Run: `make -C api openapi && npm --prefix web run gen:api`
Expected: `web/src/lib/core/api/openapi.json` and `web/src/lib/core/api/openapi.d.ts` updated cleanly (`Format: uuid` on required UUID fields; no `format: "uuid"` on optional empty-string-capable fields or PCG/CERFA `line_id` slugs).

- [ ] **Step 2: Write failing unit tests for `formatShortId` in `web/src/lib/core/formatters/id.test.ts`**

```ts
import { describe, expect, it } from 'vitest';
import { formatShortId } from './id';

describe('formatShortId', () => {
	it('extracts trailing random hex digits so same-millisecond UUIDv7 values do not collide', () => {
		const idA = '0192dce0-7c4a-7001-8abc-111122223333';
		const idB = '0192dce0-7c4a-7002-9def-444455556666';

		expect(formatShortId(idA)).toBe('22223333');
		expect(formatShortId(idB)).toBe('55556666');
		expect(formatShortId(idA, 6)).toBe('223333');
		expect(formatShortId(idB, 6)).toBe('556666');
		expect(formatShortId(idA, 6)).not.toBe(formatShortId(idB, 6));
	});

	it('strips hyphens and handles short or empty strings safely', () => {
		expect(formatShortId('  abc-12  ', 8)).toBe('abc12');
		expect(formatShortId('', 8)).toBe('');
	});
});
```

- [ ] **Step 3: Run Vitest to verify RED**

Run: `npm --prefix web test`
Expected: FAIL (`Cannot find module './id'`).

- [ ] **Step 4: Implement `formatShortId`, replace all 8 UI ID-prefix `.slice(...)` calls, and update Vitest UUIDv7 fixtures**

1. Create `web/src/lib/core/formatters/id.ts`:
   ```ts
   export function formatShortId(id: string, length = 8): string {
   	const compact = id.trim().replace(/-/g, '');
   	if (compact.length <= length) {
   		return compact;
   	}
   	return compact.slice(-length);
   }
   ```
2. Replace all 8 entity-ID `.slice(0, 6)` / `.slice(0, 8)` calls with `formatShortId(..., 6)` / `formatShortId(..., 8)` in:
   - `web/src/routes/(app)/inspections/(components)/CreateInspectionDialog.svelte` (`formatShortId(lease.id, 6)` and `formatShortId(entryInsp.id, 8)`)
   - `web/src/routes/(app)/inspections/(components)/InspectionsListCard.svelte` (`formatShortId(lease.id, 6)` and `formatShortId(inspection.lease_id, 8)`)
   - `web/src/routes/(app)/inspections/incidents/(components)/IncidentsTableCard.svelte` (`formatShortId(lease.id, 6)`)
   - `web/src/routes/(app)/inspections/incidents/(components)/CreateIncidentDialog.svelte` (`formatShortId(lease.id, 6)`)
   - `web/src/routes/(app)/rents/(components)/rent-rows.ts` (`Lot ${formatShortId(lease.lot_id, 8)}`)
   - `web/src/routes/(app)/leases/(components)/LeasesTableCard.svelte` (`Lot {formatShortId(lease.lot_id, 8)}`)
   *(Leave ISO date `.slice(0, 10)`, SIREN `.slice(0, 9)`, array `.slice(0, 3)`, and initials `.slice(0, 2)` untouched.)*
3. Update entity/lease/lot/tenant/user ID fixtures in `web/src/routes/(app)/rents/(components)/rent-rows.test.ts` (including adding an assertion on fallback `Lot <trailing-8-hex>` when a lot is absent from `propertiesByEntity`), `web/src/lib/modules/cockpit/cockpit-builder.test.ts`, `web/src/lib/core/workspace/onboarding-gate.test.ts`, and `web/src/lib/server/backend-api.test.ts` from legacy `'ent_1'` / `'lease_1'` strings to canonical RFC 9562 `UUIDv7` strings (preserving `'ALL'` for `selectedEntityId` and PCG/CERFA statement `line_id` slugs in `statement-state.test.ts`).

- [ ] **Step 5: Run mandatory formatting, linting, tests, typecheck, build, and Go suite**

Run:
1. `npm --prefix web run format`
2. `npm --prefix web run lint && npm --prefix web test && npm --prefix web run check && npm --prefix web run build`
3. `make -C api test && make -C api lint && make -C api build`
Expected: PASS with 0 errors and 0 warnings across both `web/` and `api/`.

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/core/api/openapi.json web/src/lib/core/api/openapi.d.ts web/src/lib/core/formatters/id.ts web/src/lib/core/formatters/id.test.ts web/src/routes/ web/src/lib/modules/cockpit/cockpit-builder.test.ts web/src/lib/core/workspace/onboarding-gate.test.ts web/src/lib/server/backend-api.test.ts
git commit -m "feat(web): sync UUID OpenAPI schemas and use trailing random hex in formatShortId"
```
