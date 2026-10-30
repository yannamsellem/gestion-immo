# PostgreSQL 18 & Native RFC 9562 UUIDv7 Identity Architecture — Design Specification

- **Date:** 2026-10-30
- **Status:** Pending User Approval (Builder Check Completed)
- **Scope:** Docker Infrastructure (`postgres:18-alpine`), Greenfield `tern` SQL Schema (`001`–`006`), Go Platform Packages (`internal/platform/id`, `internal/platform/secret`, `internal/platform/postgres`), All 7 Go Bounded Context Modules (`auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, `accounting`), OpenAPI 3.1 (`Huma v2`) + TypeScript Codegen, and SvelteKit Short-ID Display Formatting.

---

## 1. Intent & Why

Currently, every aggregate root and entity across `gestion-immo` generates primary keys as `<prefix>_<32-random-hex-chars>` (`ent_...`, `lse_...`, `edl_...`, `edli_...`) stored in PostgreSQL `TEXT` columns:

1. **B-Tree Index Fragmentation & Write Amplification:** Purely random hex payloads after a constant string prefix scatter inserts across B-tree leaf pages, causing frequent page splits, poor buffer-cache locality, and index bloat as tables (`rent_calls`, `payments`, `rent_allocations`, `journal_entries`, `journal_lines`, `inspection_items`) grow.
2. **Storage & Join Overhead:** Variable-length `TEXT` keys (`36–40 bytes` + `varlena` header) occupy more than `2.3x` the space of a fixed 16-byte native PostgreSQL `UUID` across primary keys, secondary composite indexes, and every foreign-key column.
3. **Weak Referential & Type Integrity in SQL:** Storing IDs as `TEXT` prevented database-level UUID validation and led to pseudo-nullable `TEXT NOT NULL DEFAULT ''` foreign-key anti-patterns (`stored_files.legal_entity_id`, `stored_files.lease_id`, `gli_dossiers.template_id`, `incidents.property_id`, `incidents.lot_id`, `incidents.lease_id`) without `REFERENCES` constraints.
4. **Domain Language & SRP Leaks:** Mixing entity identities, opaque bearer secrets (`mcp_live_...`, `rt_live_...`), and human receipt suffixes (`rcu_...`) inside `internal/platform/id.Generate(prefix)` violated the Single Responsibility Principle (SRP) and leaked French acronyms (`"edl"`, `"edli"` in `inspection`) into code in violation of `AGENTS.md` Rule 1.

Because entity identifiers are exposed across REST URLs (`/api/v1/entities/{id}`), browser cookies (`gi_entity_id`), SvelteKit form payloads, JWT `sub` claims, R2 object paths, and MCP tool arguments, sequential `BIGINT GENERATED ALWAYS AS IDENTITY` integers would introduce enumeration/IDOR surface unless wrapped in a reversible obfuscation layer (e.g., Sqids) across every boundary.

**Chosen Architecture:** Standardize all entity primary and foreign keys on **RFC 9562 `UUIDv7`** stored as native 16-byte **`UUID`** columns on **PostgreSQL 18 (`postgres:18-alpine`)**:
- **48-bit Unix millisecond timestamp prefix** ensures monotonic B-tree append locality comparable to `BIGINT IDENTITY`.
- **74 bits of CSPRNG entropy (`crypto/rand`)** prevent ID enumeration across REST, cookies, and MCP without any obfuscation layer.
- **DDD / Clean Architecture birth-time identity:** Go domain/application services mint `UUIDv7` values in memory via `id.NewV7()` before persistence (preserving pre-insert R2 object-key composition, Unit-of-Work batching, and Liskov Substitution Principle parity between `memory` and `postgres` repositories), while PostgreSQL 18 enforces native `UUID PRIMARY KEY DEFAULT uuidv7()` at the schema layer.
- **Pre-deployment greenfield rewrite:** Because `gestion-immo` has not been deployed to production yet, all `tern` migrations are rewritten cleanly from scratch with zero legacy `ALTER COLUMN` migration debt.

---

## 2. Goals, Non-Goals, and Anti-Goals

### Goals
1. **Upgrade PostgreSQL to `postgres:18-alpine`** for both `postgres` and `pg-backup` services in `api/docker-compose.yml` (adopting PostgreSQL 18's `/var/lib/postgresql` container volume mount path) and update project documentation (`README.md`).
2. **Rewrite `api/migrations/` cleanly (`001` through `006`, removing `007`)** so all 24 SQL tables use native `UUID PRIMARY KEY DEFAULT uuidv7()`, all optional entity links become real nullable `UUID REFERENCES ... ON DELETE SET NULL` columns, every foreign-key column is indexed, and dead `pcg_accounts` DDL is eliminated.
3. **Replace `id.Generate(prefix)` with `github.com/google/uuid` (`v1.6.0+`) encapsulated behind `id.NewV7()`, `id.MustNewV7()`, and `id.IsValidUUID()`** in `api/internal/platform/id` (providing sub-millisecond monotonic RFC 9562 `UUIDv7` generation while keeping domain/application layers decoupled from external packages).
4. **Enforce SRP for non-entity tokens and human references:**
   - Move bearer token secret generation to `api/internal/platform/secret` using 32 random bytes (`256` bits / `64` hex chars via idiomatic `make([]byte, 32)`): unprefixed `secret.GenerateToken()` for opaque HTTP refresh tokens, and prefixed `secret.GeneratePrefixedToken("mcp_live")` for human-configured MCP Personal Access Tokens.
   - Generate human-readable cash receipt suffixes (`CAISSE-YYYYMMDD-<6-hex>`) locally inside `api/internal/modules/payment/application` without coupling to `internal/platform/id`.
5. **Guarantee LSP & HTTP 4xx Error Semantics (`22P02` / `UUIDCodec` safety):** Ensure malformed UUID inputs never reach `pgx/v5`'s binary `UUIDCodec` or trigger PostgreSQL SQLSTATE `22P02` as HTTP `500 Internal Server Error`, behaving identically across `memory` and `postgres` repositories.
6. **Synchronize OpenAPI 3.1 & Frontend (`web/`):** Annotate required UUID parameters and DTO fields with `format:"uuid"`, regenerate `web/src/lib/core/api/openapi.json` and `web/src/lib/core/api/openapi.d.ts`, and replace timestamp-prefix `.slice(0, 6)` / `.slice(0, 8)` UI truncations with a trailing-random-hex `formatShortId` helper.

### Non-Goals
- Writing in-place `ALTER TABLE ... ALTER COLUMN ... TYPE UUID` data-migration scripts for pre-existing PostgreSQL 16 volumes (greenfield pre-deployment reset).
- Converting business codes or domain slugs (`account_code`, `journal_code`, `siret`, `receipt_number`, PCG/CERFA statement slugs `line_id` / `disabled_line_ids`, or the SvelteKit `'ALL'` entity filter cookie sentinel) into UUIDs.

### Anti-Goals (Failure Conditions Even If Happy-Path Tests Pass)
- Leaking an HTTP `500 Internal Server Error` (either via `pgx/v5` client-side `cannot parse UUID` or PostgreSQL `22P02`) when a caller supplies a malformed ID such as `"invalid"` or `""`.
- Breaking optional-ID REST endpoints (e.g., `POST /api/v1/leases/{id}/gli-dossier` sending `{"template_id": ""}`, `POST /api/v1/incidents` with omitted `lot_id`/`lease_id`, or `GET /api/v1/incidents` without query filters) by placing Huma `format:"uuid"` on plain `string` fields that legitimately carry `""` when unset.
- Leaving `id.Generate` or legacy `<prefix>_<hex>` entity IDs anywhere in production code or tests.
- Truncating `UUIDv7` strings with `.slice(0, 6)` or `.slice(0, 8)` in the UI (which displays only the shared millisecond/hour timestamp prefix and makes records created in the same afternoon visually identical).
- Embedding `tern` migrations inside the Go binary (`AGENTS.md` Rule 4) or modifying `web/src/lib/components/ui/**`.

---

## 3. Architectural & Clean Code Constraints (`AGENTS.md`)

1. **100% English Identifiers (`AGENTS.md` Rule 1):** Every Go identifier, SQL table/column/index/constraint name, OpenAPI field, and TypeScript symbol must be strictly English. Removing `"edl"` and `"edli"` prefixes in `inspection` resolves an existing French-acronym leak.
2. **Integer Monetary Arithmetic (`AGENTS.md` Rule 2):** Untouched (`money.AmountCents`, `money.BasisPoints`).
3. **Explicit Modular Monolith DI & CLI SRP (`AGENTS.md` Rules 3 & 4):** No new cross-module coupling; `tern` remains external (`Makefile` and Docker `migrate` target).
4. **SOLID (`AGENTS.md` Rule 5):**
   - **SRP:** `internal/platform/id` handles exclusively RFC 9562 UUID identity generation and validation; `internal/platform/secret` handles redacted secrets and cryptographic bearer token generation.
   - **LSP:** `infrastructure/memory` and `infrastructure/postgres` repositories must return identical domain errors (`Err...NotFound`, empty slices, `ErrValidation`) for all intra-module operations and all malformed-UUID inputs; for cross-module SQL `REFERENCES` constraints that only exist in PostgreSQL (`SQLSTATE 23503`), `postgres` write repositories map foreign-key violations deterministically to `domain.ErrValidation` (`422 Unprocessable Entity`) rather than leaking `500 Internal Server Error` *(My Call)*.
   - **Zero Dead Code & Mandatory Tooling:** Delete `007_schema_constraints_and_indexes.sql`, `id.Generate`, and `isSafeScopeID`; run `npm --prefix web run format` after any `web/` edit and pass full `api` + `web` verification suites.

---

## 4. Detailed Architecture & Technical Design

### 4.1 Docker & PostgreSQL 18 Infrastructure

1. **`api/docker-compose.yml`:**
   - Upgrade `services.postgres.image` from `postgres:16-alpine` to **`postgres:18-alpine`**.
   - Update `services.postgres.volumes` from `pg_data:/var/lib/postgresql/data` to **`pg_data:/var/lib/postgresql`**.
     - *Rationale:* Official Docker `postgres:18` images changed `PGDATA` to `/var/lib/postgresql/18/docker` and declare `VOLUME /var/lib/postgresql` so major-version subdirectories live cleanly under `/var/lib/postgresql`. Mounting `/var/lib/postgresql/data` directly is deprecated/incompatible with the new layout.
   - Upgrade `services.pg-backup.image` from `postgres:16-alpine` to **`postgres:18-alpine`** so `pg_dump` in `scripts/backup-to-r2.sh` matches PostgreSQL 18 server catalog versions.
2. **`README.md`:**
   - Update the stack description from `PostgreSQL 16` to `PostgreSQL 18` (noting native `UUIDv7` primary keys) and note `docker compose -f api/docker-compose.yml down -v` when reinitializing a local pre-18 Docker volume.

---

### 4.2 Go Platform Packages (`id`, `secret`, `postgres`)

#### 4.2.1 `api/internal/platform/id` (RFC 9562 `UUIDv7` via `github.com/google/uuid`)
Add `github.com/google/uuid` (`v1.6.0+`, zero transitive dependencies) to `api/go.mod`, encapsulate it strictly inside `api/internal/platform/id`, remove `Generate(prefix string)` completely, and expose:
- **`NewV7() (string, error)`**:
  - Delegates to `uuid.NewV7()` (providing RFC 9562 48-bit Unix millisecond timestamp + sub-millisecond monotonic sequencing + CSPRNG entropy) and returns `u.String()` (canonical lowercase 36-character `8-4-4-4-12` format).
- **`MustNewV7() string`**:
  - Calls `NewV7()` and panics only if the OS CSPRNG fails; convenient for unit/integration test fixtures.
- **`IsValidUUID(s string) bool`**:
  - Enforces strict canonical 36-character `8-4-4-4-12` UUID syntax: `len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' && uuid.Validate(s) == nil` (rejecting non-canonical `urn:uuid:...`, `{...}`, or 32-char hyphenless forms that `uuid.Validate` alone accepts, as well as empty strings, whitespace, or legacy prefixed IDs `ent_...`).
- **Unit tests (`api/internal/platform/id/id_test.go`):**
  - Verify 36-char canonical format, version nibble `'7'` at index `14`, variant nibble `{'8','9','a','b'}` at index `19`, strict monotonic lexicographical ordering across rapid consecutive calls within the same millisecond, uniqueness across 10,000 generated IDs, and `IsValidUUID` table-driven edge cases (including rejection of `urn:uuid:...`, `{...}`, and 32-char hex).

#### 4.2.2 `api/internal/platform/secret` (Bearer & API Token Generation)
Add `api/internal/platform/secret/token.go` (+ `secret_test.go`) using idiomatic slice allocation (`bytes := make([]byte, 32)` → 256 bits of `crypto/rand` entropy encoded as 64 lowercase hex characters via `hex.EncodeToString(bytes)`):
- **`GenerateToken() (string, error)`**:
  - Generates an unprefixed 64-character hex cryptographic secret (`32` random bytes).
  - Used by `api/internal/modules/auth/application/service.go` for opaque HTTP refresh tokens (which live strictly in `HttpOnly` cookies / auth refresh payloads and do not need a human prefix).
- **`GeneratePrefixedToken(prefix string) (string, error)`**:
  - Validates `strings.TrimSpace(prefix) != ""`, calls `GenerateToken()`, and returns `fmt.Sprintf("%s_%s", trimmedPrefix, raw)`.
  - Used exclusively by `api/internal/modules/auth/application/service.go` for `secret.GeneratePrefixedToken("mcp_live")` (human-configured MCP Personal Access Tokens where a recognizable secret-scanning prefix is valuable).

#### 4.2.3 `api/internal/platform/postgres` (Nullable UUID & `PgError` Helpers)
Add `api/internal/platform/postgres/uuid.go` (builder's call on exact filename inside `internal/platform/postgres`):
- **`NullableUUID(s string) *string`**:
  - Trims whitespace; returns `nil` when `s == ""` and `&trimmed` otherwise.
  - Allows `pgx/v5` to bind `nil` as SQL `NULL` and non-empty strings directly via its binary 16-byte `UUIDCodec` without `NULLIF($1, '')::uuid` text casts.
- **`DerefString(p *string) string`**:
  - Returns `""` when `p == nil` and `*p` otherwise.
  - Allows `rows.Scan` / `QueryRow.Scan` to scan nullable `UUID` columns directly into `*string` in binary protocol mode and hydrate Go domain structs expecting `""` when unset.
- **`IsForeignKeyViolation(err error) bool`** *(My Call)*:
  - Inspects `*pgconn.PgError` via `errors.As` and returns `true` when `pgErr.Code == "23503"` (`foreign_key_violation`), enabling DRY translation of missing parent UUID references into module `domain.ErrValidation` (`422 Unprocessable Entity`) across all PostgreSQL write repositories without introducing new cross-module DI ports.

---

### 4.3 Greenfield PostgreSQL 18 Schema (`api/migrations/001..006`)

Consolidate all constraints and indexes from `007_schema_constraints_and_indexes.sql` directly into `002`, `003`, and `005`, omit the unused `pcg_accounts` table from `006`, and **delete `007_schema_constraints_and_indexes.sql`**.

Every entity table uses `id UUID PRIMARY KEY DEFAULT uuidv7()`.

#### Complete 20-Table UUID Column & Foreign-Key Matrix

| Migration | Table | Primary Key | `UUID` Foreign Keys & `UUID` Correlation Columns | Non-UUID `TEXT` / `TEXT[]` Identifiers Preserved |
|-----------|-------|-------------|--------------------------------------------------|--------------------------------------------------|
| `001_auth_and_users.sql` | `users` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `approved_by UUID REFERENCES users(id) ON DELETE SET NULL` (+ partial index `WHERE approved_by IS NOT NULL`) | `email`, `password_hash`, `role`, `status` |
| `001_auth_and_users.sql` | `refresh_tokens` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`, `family_id UUID NOT NULL`, `replaced_by_id UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL` | `token_hash` (SHA-256 hex) |
| `001_auth_and_users.sql` | `mcp_api_tokens` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE` | `label`, `token_hash` (SHA-256 hex) |
| `002_portfolio_hierarchy.sql` | `legal_entities` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | — | `siret`, `naf_code`, `entity_type` |
| `002_portfolio_hierarchy.sql` | `shareholders` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE` | `full_name` |
| `002_portfolio_hierarchy.sql` | `properties` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE` | `property_type` (with inline `CHECK` from former `007`), `accounting_category` |
| `002_portfolio_hierarchy.sql` | `lots` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `property_id UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE` | `lot_type` (`DEFAULT 'APARTMENT'` + inline `CHECK` from former `007`) |
| `003_leases_and_tenants.sql` | `tenants` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | — | `full_name`, `email`, `phone` |
| `003_leases_and_tenants.sql` | `leases` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE`, `lot_id UUID NOT NULL REFERENCES lots(id) ON DELETE RESTRICT`, `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT` | `lease_type`, `gli_policy_number` |
| `003_leases_and_tenants.sql` | `rent_calls` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE`, `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE`, `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT` (+ `idx_rent_calls_tenant_id` & `idx_rent_calls_lease_unpaid_order` from former `007`) | `period_month` (`YYYY-MM`), `status` |
| `004_payments_and_cash_journal.sql` | `payments` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE`, `lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE RESTRICT`, `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT` | `method`, `external_ref`, `receipt_number` |
| `004_payments_and_cash_journal.sql` | `rent_allocations` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE`, `rent_call_id UUID NOT NULL REFERENCES rent_calls(id) ON DELETE RESTRICT` | `period_month` |
| `004_payments_and_cash_journal.sql` | `cash_journal_entries` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE`, `payment_id UUID REFERENCES payments(id) ON DELETE SET NULL` (+ partial index `WHERE payment_id IS NOT NULL`) | `account_code`, `counterpart_account_code`, `entry_type`, `receipt_number` |
| `005_templates_inspections_incidents.sql` | `document_templates` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | — | `template_type`, `format`, `r2_key`, `html_body` |
| `005_templates_inspections_incidents.sql` | `stored_files` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID REFERENCES legal_entities(id) ON DELETE SET NULL`, `lease_id UUID REFERENCES leases(id) ON DELETE SET NULL` | `category`, `file_name`, `content_type`, `r2_key` |
| `005_templates_inspections_incidents.sql` | `gli_dossiers` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE`, `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE`, `template_id UUID REFERENCES document_templates(id) ON DELETE SET NULL` (+ `idx_gli_dossiers_template_id`) | `gli_policy_number`, `unpaid_months TEXT[]`, `archive_r2_key`, `included_files TEXT[]` |
| `005_templates_inspections_incidents.sql` | `inspection_templates` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | — | `name`, `rooms TEXT[]`, `default_elements TEXT[]` |
| `005_templates_inspections_incidents.sql` | `inspections` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE` (inlined from former `007`), `template_id UUID NOT NULL REFERENCES inspection_templates(id) ON DELETE RESTRICT`, `entry_inspection_id UUID REFERENCES inspections(id) ON DELETE SET NULL` (+ partial index `WHERE entry_inspection_id IS NOT NULL`) | `inspection_type`, `report_pdf_r2_key` |
| `005_templates_inspections_incidents.sql` | `inspection_items` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `inspection_id UUID NOT NULL REFERENCES inspections(id) ON DELETE CASCADE` | `room_name`, `element_name`, `condition`, `photo_r2_keys TEXT[]` |
| `005_templates_inspections_incidents.sql` | `incidents` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `property_id UUID REFERENCES properties(id) ON DELETE SET NULL`, `lot_id UUID REFERENCES lots(id) ON DELETE SET NULL` (+ `idx_incidents_lot_id`), `lease_id UUID REFERENCES leases(id) ON DELETE SET NULL` | `title`, `status`, `insurance_claim_number`, `observations TEXT[]`, `photo_r2_keys TEXT[]` |
| `006_accounting_and_annual_accounts.sql` | `journal_entries` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE` | `journal_code`, `entry_number`, `document_ref` |
| `006_accounting_and_annual_accounts.sql` | `journal_lines` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE`, `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE` | **`line_id TEXT NOT NULL DEFAULT ''`** (PCG/CERFA statement slug, e.g., `'PRODUITS_LOYERS'`), `account_code`, `sub_account_code`, `reconciliation_code`, `currency_code` |
| `006_accounting_and_annual_accounts.sql` | `account_balance_lines` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE` | **`line_id TEXT NOT NULL`** (PCG/CERFA statement slug in `UNIQUE (legal_entity_id, fiscal_year, line_id)`), `account_code`, `section`, `cerfa_code` |
| `006_accounting_and_annual_accounts.sql` | `annual_account_reports` | `id UUID PRIMARY KEY DEFAULT uuidv7()` | `legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE` | **`disabled_line_ids TEXT[] NOT NULL DEFAULT '{}'`** (PCG/CERFA statement slugs), `pdf_r2_key`, `file_name` |

*(Note: That totals 24 SQL tables across `001`–`006` — 20 aggregate/entity tables plus 4 child-line tables `rent_allocations`, `inspection_items`, `journal_lines`, `account_balance_lines` — all 24 carrying `id UUID PRIMARY KEY DEFAULT uuidv7()`, with `pcg_accounts` removed.)*

---

### 4.4 Application Services, Repository LSP Contract & `pgx/v5` UUID Safeguards

#### 4.4.1 Entity ID, Secret, and Receipt Generation Call Sites
1. **28 Entity/Child-Line ID Call Sites → `id.NewV7()`:**
   - `auth`: `userID` (`2` sites), `mcpTokenID` (`1` site), `refreshFamilyID` (`1` site), `refreshRecordID` (`1` site).
   - `portfolio`: `entityID`, `shareholderID`, `propertyID`, `lotID`.
   - `lease`: `tenantID`, `leaseID`, `rentCallID`.
   - `payment`: `paymentID`, `allocationID`, `cashJournalEntryID` (`2` sites).
   - `document`: `templateID`, `storedFileID` (`2` sites), `gliDossierID`.
   - `inspection`: `templateID`, `inspectionID` (replaces `"edl"`), `inspectionItemID` (replaces `"edli"`), `incidentID`.
   - `accounting`: `journalEntryID`, `journalLineID`, `annualAccountReportID`, plus `AccountBalanceRepo` / `memory.AccountBalanceRepo` when `line.ID == ""` (replace `fmt.Sprintf("%s_%d_%s", entityID, fiscalYear, line.LineID)` with `id.NewV7()` so new balance lines receive a valid `UUIDv7`, while `ON CONFLICT (legal_entity_id, fiscal_year, line_id) DO UPDATE` keeps the existing row's `id` intact).
2. **2 Bearer Secret Call Sites (`auth/application/service.go`) → `internal/platform/secret`:**
   - `secret.GeneratePrefixedToken("mcp_live")` (MCP Personal Access Token)
   - `secret.GenerateToken()` (unprefixed 256-bit HTTP refresh token; also update sample string in `auth/infrastructure/security_test.go`)
3. **1 Cash Receipt Number Call Site (`payment/application/service.go`) → Local Helper:**
   - Extract `generateCashReceiptSuffix() (string, error)` inside `payment/application` using `bytes := make([]byte, 3)` (`crypto/rand`) encoded as `6` uppercase hex characters (`CAISSE-YYYYMMDD-A1B2C3`), decoupling human receipt numbers from `internal/platform/id`.
4. **R2 Scope Validation, Default GLI Template Sentinel Removal, and Pre-Upload Entity Check:**
   - In `document/application/service.go`: replace `isSafeScopeID(leaseID)` and `isSafeScopeID(entityID)` with `id.IsValidUUID(leaseID)` and `id.IsValidUUID(entityID)` and delete the dead `isSafeScopeID` helper (`AGENTS.md` Rule 5).
   - In `document/application/service.go` (`resolveGLIDocxTemplate`) *(My Call)*:
     - When `templateID == ""` (omitted), render the built-in minimal GLI `.docx` and return `resolvedTemplateID = ""` (`SQL NULL` in `gli_dossiers.template_id`), removing the legacy non-UUID `"tpl_gli_default"` sentinel.
     - When `templateID != ""`, validate `id.IsValidUUID(trimmed)` (`domain.ErrValidation` → `422`) and propagate `domain.ErrTemplateNotFound` (`404 Not Found`, matching `GenerateFromTemplate`) instead of silently falling back and persisting a non-existent template UUID that would violate `gli_dossiers.template_id REFERENCES document_templates(id)`.
   - In `accounting/application/service.go` (`GenerateAnnualAccountsPDF`) *(My Call)*: when the already-injected `s.portfolioSvc != nil`, verify the legal entity exists via `s.portfolioSvc` **before** rendering/uploading the PDF to R2 (matching `document.AttachFileToR2` and avoiding orphaned R2 objects while keeping isolated `s.portfolioSvc == nil` unit tests intact).

#### 4.4.2 Liskov Substitution Principle (LSP) & Preventing `pgx/v5` Binary `UUIDCodec` / `22P02` / `23503` Errors
In `pgx/v5`, PostgreSQL's extended query protocol types parameters compared to `UUID` columns as OID `2950` (`uuid`) or `2951` (`uuid[]`). Passing a non-UUID string (such as `"unknown"` or `""`) to `Query` / `QueryRow` / `Exec` causes `pgx/v5`'s client-side `UUIDCodec.PlanEncode` to fail **before** executing the query, which would bubble up as an unhandled `500 Internal Server Error` if unguarded. Likewise, passing a syntactically valid UUID that references a non-existent cross-module parent row triggers PostgreSQL `SQLSTATE 23503` (`foreign_key_violation`).

To guarantee clean `4xx` domain error semantics without adding new cross-module DI ports:

1. **Application Service Boundary Validation (`ErrValidation` → HTTP `422`):**
   - Every service command/method receiving a **required** entity ID (`LegalEntityID`, `PropertyID`, `LotID`, `TenantID`, `LeaseID`, `TemplateID`, `InspectionID`, `IncidentID`, `UserID`, `ApproverID`, `TokenID`, etc.) validates `id.IsValidUUID(trimmedID)` and returns `fmt.Errorf("%w: invalid <field_name>", domain.ErrValidation)` when invalid (except `Get*ByID` lookup methods where returning `Err...NotFound` → `404` on unknown/non-UUID IDs is standard REST resource lookup behavior, or when validated via Huma `format:"uuid"` at the HTTP edge — see §4.5).
   - Every service command/method receiving an **optional** entity ID (`AttachFileCmd.LegalEntityID` / `LeaseID`, `AssembleGLIDossier.templateID`, `CreateInspectionCmd.EntryInspectionID`, `CreateIncidentCmd.PropertyID` / `LotID` / `LeaseID`, `ListIncidents` filters, `ListLeases` `entityID`) validates: if `trimmedID != ""` and `!id.IsValidUUID(trimmedID)`, return `fmt.Errorf("%w: invalid <field_name>", domain.ErrValidation)`.

2. **PostgreSQL Repository Guards (Defense-in-Depth & Direct Repo Caller LSP):**
   - **`Find*ByID(ctx, rawID)` / single-ID mutations (`ArchiveTenant`, `RevokeMCPToken`, `RotateRefreshToken`, etc.):** If `!id.IsValidUUID(rawID)`, immediately return `Err...NotFound` without issuing a database round-trip (identical to an in-memory map miss).
   - **`List*By<EntityID>(ctx, rawID)`:** If `!id.IsValidUUID(rawID)`, immediately return `[]T{}, nil` without issuing a database round-trip (identical to an in-memory slice filter miss).
   - **`LotRepo.ListLotsByPropertyIDs(ctx, propertyIDs []string)` & child batch loaders (`ANY($1)`):** Filter `propertyIDs` down to `id.IsValidUUID` entries; if none remain, return `map[string][]domain.Lot{}, nil` immediately. On valid UUID slices, `WHERE property_id = ANY($1::uuid[])` works natively with `pgx/v5`.
   - **Foreign-Key Violations (`SQLSTATE 23503`) on Write Repositories *(My Call)*:** Every PostgreSQL `Create*` / `Save*` / `Upsert*` repository method checks `postgres.IsForeignKeyViolation(err)` and translates `23503` into that module's `fmt.Errorf("%w: referenced record does not exist", domain.ErrValidation)` (HTTP `422 Unprocessable Entity`), keeping `NewModule` constructors decoupled while ensuring non-existent cross-module parent UUIDs never produce HTTP `500`.
   - **`IncidentRepo.ListIncidents(ctx, propertyID, leaseID string)`:**
     - If `(propertyID != "" && !id.IsValidUUID(propertyID)) || (leaseID != "" && !id.IsValidUUID(leaseID))`, return `[]domain.Incident{}, nil`.
     - Replace `WHERE ($1 = '' OR property_id = $1) AND ($2 = '' OR lease_id = $2)` (which breaks on `UUID` columns!) with:
       ```sql
       WHERE ($1::uuid IS NULL OR property_id = $1)
         AND ($2::uuid IS NULL OR lease_id = $2)
       ```
       passing `postgres.NullableUUID(propertyID)` and `postgres.NullableUUID(leaseID)`.
   - **Nullable `UUID` Columns in `StoredFileRepo`, `GLIDossierRepo`, and `IncidentRepo`:**
     - On `INSERT` / `UPSERT`: bind optional `string` domain fields via `postgres.NullableUUID(...)` (`""` → SQL `NULL`).
     - On `SELECT`: scan nullable `UUID` columns into `*string` local pointers and assign `postgres.DerefString(...)` (`SQL NULL` → `""`), preserving the exact `domain.StoredFile`, `domain.GLIDossier`, and `domain.Incident` Go struct field types.

---

### 4.5 OpenAPI 3.1 (`Huma v2`), MCP Tool Schemas & SvelteKit Frontend (`web/`)

#### 4.5.1 OpenAPI 3.1 `format:"uuid"` Rules (Zero Empty-String Regression)
- **Add `format:"uuid"`** on:
  - All `path:"id"` route parameter structs across `interfaces/http/handler.go` in all modules.
  - All **required** UUID fields on request body structs (`CreatePropertyCmd`, `CreateLotCmd`, `CreateLeaseCmd`, `RecordCashPaymentCmd`, `RecordBankPaymentCmd`, `RecordCashDepositCmd`, `ImportPaymentsCSVCmd`, `GenerateDocCmd`, `AssembleGLIDossierBody.LeaseID`, `CreateInspectionCmd.LeaseID` & `TemplateID`, etc.).
  - All **always-populated** UUID fields (and nullable `*string` pointer UUID fields such as `PaymentID *string`, `ApprovedBy *string`, `EntryInspectionID *string`) on response `*DTO` structs.
- **Omit Huma `format:"uuid"`** on optional plain-`string` fields that legally carry `""` when unset:
  - Request/query inputs: `UploadDocumentBody.LegalEntityID` & `LeaseID`, `AssembleGLIDossierByLeaseBody.TemplateID`, `AssembleGLIDossierBody.TemplateID`, `CreateInspectionCmd.EntryInspectionID`, `CreateIncidentCmd.PropertyID` / `LotID` / `LeaseID`, and optional `query` filter parameters (`entity_id`, `lease_id`, `property_id`, `template_id`).
  - Response DTOs: `StoredFileDTO.LegalEntityID` & `LeaseID`, `GLIDossierArchiveDTO.TemplateID` (if exposed), `IncidentDTO.PropertyID` / `LotID` / `LeaseID`, and `JournalLineDTO.LineID` / `AccountBalanceLineDTO.LineID` / `DisabledLineIDs` (which are PCG/CERFA statement slugs, not UUIDs).
  - *Rationale (My Call):* Huma's JSON Schema validator rejects `""` when `format:"uuid"` is attached to a non-pointer `string` field, whereas `web/src/lib/server/backend-api.ts` legitimately sends `{ template_id: templateId ?? '' }` or omits optional filters. Application services validate non-empty values with `id.IsValidUUID` and return `domain.ErrValidation` (`422 Unprocessable Entity`).
- **Regenerate OpenAPI & TypeScript Definitions:**
  - Run `make -C api openapi` → updates `web/src/lib/core/api/openapi.json`.
  - Run `npm --prefix web run gen:api` → updates and formats `web/src/lib/core/api/openapi.d.ts`.

#### 4.5.2 MCP Tool Descriptions (`interfaces/mcp`)
- Ensure MCP tool parameter descriptions across modules refer cleanly to UUID identifiers (no stale hints suggesting `ent_...` or `lse_...` prefixes if any exist).

#### 4.5.3 SvelteKit Short-ID Display Formatter & Cookie Handling (`web/`)
- **Why `.slice(0, 6)` / `.slice(0, 8)` Must Change:** In `UUIDv7`, the leading 12 hex digits encode the 48-bit Unix millisecond timestamp. Truncating the prefix (`lease.id.slice(0, 6)`) produces identical labels (`Bail 0192dc`) for every record created within the same ~4.6-hour window. The trailing 12 hex digits (`rand_b`) are pure CSPRNG random hex with no hyphens.
- **Shared Formatter (`web/src/lib/core/formatters/id.ts` + `id.test.ts`):**
  - Export `formatShortId(id: string, length = 8): string` which strips hyphens (`id.trim().replace(/-/g, '')`), returns the trailing `length` characters in uppercase or lowercase (keep lowercase hex to match UUID convention: `.slice(-length)`), and handles strings shorter than `length` gracefully.
  - Replace all 7 `.slice(0, 6)` / `.slice(0, 8)` entity-ID fallback/disambiguator occurrences in:
    1. `web/src/routes/(app)/inspections/(components)/CreateInspectionDialog.svelte` (2 sites: lease label & move-in inspection label)
    2. `web/src/routes/(app)/inspections/(components)/InspectionsListCard.svelte` (2 sites)
    3. `web/src/routes/(app)/inspections/incidents/(components)/IncidentsTableCard.svelte` (1 site)
    4. `web/src/routes/(app)/inspections/incidents/(components)/CreateIncidentDialog.svelte` (1 site)
    5. `web/src/routes/(app)/rents/(components)/rent-rows.ts` (1 site: `Lot ${formatShortId(lease.lot_id, 8)}`)
    6. `web/src/routes/(app)/leases/(components)/LeasesTableCard.svelte` (1 site: `Lot {formatShortId(lease.lot_id, 8)}`)
    *(Total: 8 call sites across 6 files; non-ID `.slice(...)` calls on ISO dates and SIREN numbers remain untouched.)*
- **Workspace Entity Cookie (`gi_entity_id`):**
  - Continues accepting either the `'ALL'` sentinel or a `UUIDv7` legal entity ID (`backend-api.ts` already guards `entityId && entityId !== 'ALL'`).
- **Test Fixtures:**
  - Update all Go test fixtures (`api/cmd/main_test.go`, `api/internal/modules/*/application/service_test.go`, `api/internal/platform/postgres/integration_test.go`) and Vitest fixtures (`web/src/**/*.test.ts`) from legacy prefixed IDs (`ent_1`, `lease_1`, `lot_1`, etc.) to valid canonical RFC 9562 `UUIDv7` strings so tests exercise real UUID validation end-to-end.

---

## 5. What Is Left to the Builder

1. Exact helper placement/naming for `NullableUUID` / `DerefString` inside `api/internal/platform/postgres` and deterministic UUIDv7 fixture constants in test files.
2. Whether `memory` repositories also normalize UUID casing via `strings.ToLower` on save/lookup or rely on `id.NewV7()` lowercase generation + case-insensitive `id.IsValidUUID` (as long as `memory` and `postgres` behave identically on valid and invalid IDs).
3. Exact error-detail wording in `fmt.Errorf("%w: invalid ...", domain.ErrValidation)` messages where no existing test asserts a specific substring.

---

## 6. Verification & Acceptance Criteria

1. **Go Unit & Race Tests:** `make -C api test`, `make -C api test-race`, `make -C api lint`, `make -C api build`.
2. **PostgreSQL 18 + `tern` Integration Verification:**
   - Start `postgres:18-alpine` (`docker compose -f api/docker-compose.yml up -d postgres`), apply `make -C api migrate-up`, verify `make -C api migrate-down` / `migrate-up` round-trip cleanly on `001`–`006`, and run `TEST_DATABASE_URL=... go -C api test -v ./internal/platform/postgres/...` covering all 24 tables, nullable `UUID` foreign keys (`stored_files`, `gli_dossiers`, `incidents`), `ANY($1::uuid[])` batch queries, and malformed-UUID `Err...NotFound` / empty-slice LSP checks.
3. **Zero Legacy ID Audit:** `grep -rn "id\.Generate" api/` returns zero matches.
4. **OpenAPI & Frontend Verification:**
   - `make -C api openapi` && `npm --prefix web run gen:api`
   - `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check && npm --prefix web run build`
