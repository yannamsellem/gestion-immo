# Plan 007: Eliminate N+1 PostgreSQL queries, batch multi-row writes, add schema constraints/indexes, and add Postgres integration test harness

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in the `infrastructure/postgres` repositories and `api/migrations/` against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: `plans/001-dx-makefile-compose-and-backup-hardening.md`, `plans/005-portfolio-lease-and-payment-correctness.md`, `plans/006-accounting-live-portfolio-and-balance-sync.md`
- **Category**: perf | migration | tests
- **Covers findings**: `PERF-01`, `PERF-02`, `DEP-02`, `TEST-01`, `TEST-02` (HTTP/MCP route coverage in `document`, `inspection`, `lease`)
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

Four PostgreSQL list paths currently issue `1 + N` queries (`ListPaymentsByLease`/`ListPaymentsByEntity` querying `rent_allocations` per payment, `ListJournalEntries` querying `journal_lines` per entry, `ListInspectionsByLease` querying `inspection_items` per inspection, and `ListProperties`/`GetEntityPortfolioSummary` querying `lots` per property), while `ListNegativePositions` issues `1 + 2N` queries across leases. Multi-row writes in `SaveRentCalls`, `UpsertInspectionItems`, `SaveJournalEntry`, and `UpsertAccountBalanceLines` execute `conn.Exec` in a loop instead of `pgx.Batch` (`conn.SendBatch`). In addition, `FindUserByEmail` uses `WHERE LOWER(email) = LOWER($1)` (defeating the unique B-tree index on `users.email`), `inspections.lease_id` and `properties.property_type`/`lots.lot_type` lack SQL foreign-key/check constraints, and none of the 19 `infrastructure/postgres` files have integration tests.

## Current state

- `api/internal/modules/payment/infrastructure/postgres/payment_repo.go:161-167`:
  ```go
  	for i := range payments {
  		allocs, err := r.listAllocationsByPayment(ctx, payments[i].ID)
  		if err != nil {
  			return nil, err
  		}
  		payments[i].Allocations = allocs
  	}
  ```
- `api/internal/modules/accounting/infrastructure/postgres/journal_entry_repo.go:142-148`:
  ```go
  	for i := range entries {
  		lines, err := r.loadJournalLines(ctx, entries[i].ID)
  		if err != nil {
  			return nil, err
  		}
  		entries[i].Lines = lines
  	}
  ```
- `api/internal/modules/inspection/infrastructure/postgres/inspection_repo.go:138-144`:
  ```go
  	for i := range out {
  		items, err := r.loadInspectionItems(ctx, out[i].ID)
  		if err != nil {
  			return nil, err
  		}
  		out[i].Items = items
  	}
  ```
- `api/internal/modules/auth/infrastructure/postgres/user_repo.go:100-108`:
  ```go
  func (r *UserRepo) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
  	const query = `
  		SELECT id, full_name, email, password_hash, role, status, approved_by, approved_at, created_at, updated_at
  		FROM users
  		WHERE LOWER(email) = LOWER($1)
  	`
  ```
- `api/migrations/000005_templates_inspections_incidents.up.sql:55-65`:
  ```sql
  CREATE TABLE IF NOT EXISTS inspections (
      id TEXT PRIMARY KEY,
      lease_id TEXT NOT NULL,
      template_id TEXT NOT NULL REFERENCES inspection_templates(id) ON DELETE RESTRICT,
  ```

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Test | `cd api && go test -v ./...` | exit 0 |

## Scope

**In scope**:

- `api/migrations/000007_schema_constraints_and_indexes.up.sql` (create)
- `api/migrations/000007_schema_constraints_and_indexes.down.sql` (create)
- `api/internal/modules/auth/infrastructure/postgres/user_repo.go`
- `api/internal/modules/portfolio/domain/uow.go`
- `api/internal/modules/portfolio/infrastructure/memory/lot_repo.go`
- `api/internal/modules/portfolio/infrastructure/postgres/lot_repo.go`
- `api/internal/modules/portfolio/application/service.go`
- `api/internal/modules/lease/infrastructure/postgres/rent_call_repo.go`
- `api/internal/modules/lease/application/service_test.go`
- `api/internal/modules/payment/infrastructure/postgres/payment_repo.go`
- `api/internal/modules/inspection/infrastructure/postgres/inspection_repo.go`
- `api/internal/modules/inspection/application/service.go`
- `api/internal/modules/inspection/application/service_test.go`
- `api/internal/modules/document/application/service_test.go`
- `api/internal/modules/accounting/infrastructure/postgres/journal_entry_repo.go`
- `api/internal/modules/accounting/infrastructure/postgres/account_balance_repo.go`
- `api/internal/platform/postgres/integration_test.go` (create)

**Out of scope**:

- Existing migration files `000001`..`000006` (add changes via `000007` so migration history stays monotonic).

## Git workflow

- Branch: `advisor/007-postgres-migrations-batching-and-n-plus-one`
- Commit message style: `perf(postgres): eliminate N+1 queries, batch writes, add migration 000007 and integration tests`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Create migration `000007_schema_constraints_and_indexes.{up,down}.sql` and fix `FindUserByEmail` (`DEP-02`, `PERF-02`)

1. Create `api/migrations/000007_schema_constraints_and_indexes.up.sql` (and matching `.down.sql`):
   - Add foreign key `ALTER TABLE inspections ADD CONSTRAINT fk_inspections_lease FOREIGN KEY (lease_id) REFERENCES leases(id) ON DELETE CASCADE;`.
   - Add enum `CHECK` constraints `chk_properties_property_type` on `properties(property_type)` (`'APARTMENT'`, `'HOUSE'`, `'COMMERCIAL_PREMISES'`, `'BARE_LAND'`, `'PARKING'`, `'CELLAR'`, `'BUILDING'`) and `chk_lots_lot_type` on `lots(lot_type)`.
   - Add composite partial index `CREATE INDEX IF NOT EXISTS idx_rent_calls_lease_unpaid_order ON rent_calls(lease_id, due_date ASC, period_month ASC) WHERE remaining_due_cents > 0 AND status <> 'PAID';`.
   - Drop the unused `pcg_accounts` table (`DROP TABLE IF EXISTS pcg_accounts;`, recreating it in `.down.sql`).
2. In `api/internal/modules/auth/infrastructure/postgres/user_repo.go:100-108` (`FindUserByEmail`), change `WHERE LOWER(email) = LOWER($1)` to `WHERE email = LOWER($1)` so PostgreSQL uses the unique index on `users.email`.

**Verify**: `cd api && go vet ./...` → exit 0.

### Step 2: Eliminate N+1 child queries with `ANY($1)` and avoid redundant inspection reloads (`PERF-01`, `PERF-02`)

1. In `api/internal/modules/payment/infrastructure/postgres/payment_repo.go:148-190` (`collectPaymentsWithAllocations`):
   - When `len(payments) > 0`, collect all `paymentIDs := make([]string, len(payments))` and execute a single query `SELECT id, payment_id, rent_call_id, period_month, applied_cents, created_at FROM rent_allocations WHERE payment_id = ANY($1) ORDER BY created_at ASC`, grouping allocations into a `map[string][]domain.RentAllocation` and attaching them to `payments[i].Allocations`.
2. In `api/internal/modules/accounting/infrastructure/postgres/journal_entry_repo.go:142-170` (`ListJournalEntries`):
   - When `len(entries) > 0`, collect all `entryIDs` and execute a single query `SELECT ... FROM journal_lines WHERE journal_entry_id = ANY($1) ORDER BY id ASC`, grouping lines by `JournalEntryID`.
3. In `api/internal/modules/inspection/infrastructure/postgres/inspection_repo.go:138-146` (`ListInspectionsByLease`):
   - When `len(out) > 0`, collect all `inspectionIDs` and execute a single query `SELECT ... FROM inspection_items WHERE inspection_id = ANY($1) ORDER BY room_name ASC, element_name ASC`, grouping items by `InspectionID`.
4. In `api/internal/modules/inspection/application/service.go:353-412` (`UpsertInspectionItems`), avoid the second `FindInspectionByID` call after `UpsertInspectionItems` by merging the upserted items into `insp.Items` in memory (or returning the updated items directly).
5. In `api/internal/modules/portfolio/domain/uow.go`, `memory/lot_repo.go`, `postgres/lot_repo.go`, and `portfolio/application/service.go`, add `ListLotsByPropertyIDs(ctx context.Context, propertyIDs []string) (map[string][]domain.Lot, error)` (`WHERE property_id = ANY($1) ORDER BY created_at ASC`) and use it in `ListProperties` and `GetEntityPortfolioSummary`.

**Verify**: `cd api && go vet ./... && go test ./...` → exit 0.

### Step 3: Batch multi-row inserts/upserts with `pgx.Batch` (`SendBatch`) (`PERF-01`)

1. In `api/internal/modules/lease/infrastructure/postgres/rent_call_repo.go:57-89` (`SaveRentCalls`), queue all rent call upserts into `batch := &pgx.Batch{}` via `batch.Queue(query, ...)` and execute `br := conn.SendBatch(txCtx, batch)` followed by `for range calls { _, err := br.Exec() ... }` and `br.Close()`.
2. Apply the same `pgx.Batch` (`conn.SendBatch`) pattern to:
   - `api/internal/modules/inspection/infrastructure/postgres/inspection_repo.go:165-183` (`UpsertInspectionItems`)
   - `api/internal/modules/accounting/infrastructure/postgres/journal_entry_repo.go:72-96` (`SaveJournalEntry` lines loop)
   - `api/internal/modules/accounting/infrastructure/postgres/account_balance_repo.go:54-75` (`UpsertAccountBalanceLines`)
   - `api/internal/modules/payment/infrastructure/postgres/payment_repo.go:71-83` (`SavePayment` allocations loop)

**Verify**: `cd api && go vet ./... && go test ./...` → exit 0.

### Step 4: Add PostgreSQL integration test harness and expand HTTP/MCP tests (`TEST-01`, `TEST-02`)

1. Create `api/internal/platform/postgres/integration_test.go` that reads `os.Getenv("TEST_DATABASE_URL")` (calling `t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")` when empty), runs `RunMigrations(dbURL, "../../../migrations", "up")`, verifies `WithinTransaction` commit and rollback on error, and tests Postgres repositories across `auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, and `accounting`.
2. Expand HTTP and MCP route tests in `api/internal/modules/document/application/service_test.go`, `api/internal/modules/inspection/application/service_test.go`, and `api/internal/modules/lease/application/service_test.go` so previously untested HTTP routes (`GenerateFromTemplate`, `ListTemplates`, `PUT /api/v1/inspections/{id}/items`, `POST /api/v1/inspections/{id}/pdf`) and MCP tool calls (`create_tenant`, `create_lease`, `list_leases`) are executed and asserted.

**Verify**: `cd api && go vet ./... && go test -v ./...` → exit 0.

## Test plan

- `api/internal/platform/postgres/integration_test.go` exercises `RunMigrations` (including `000007`), `WithinTransaction` rollback/commit, `ANY($1)` batch reads, and `pgx.Batch` writes when `TEST_DATABASE_URL` is provided, and skips cleanly in offline unit-test runs.
- Expanded HTTP/MCP tests in `document`, `inspection`, and `lease` run on every `go test ./...`.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test -race ./...` exits 0
- [ ] `api/migrations/000007_schema_constraints_and_indexes.up.sql` and `.down.sql` exist
- [ ] `api/internal/platform/postgres/integration_test.go` exists and compiles/runs cleanly
- [ ] `grep -rn "SendBatch" api/internal/modules/` returns matches in `lease`, `payment`, `inspection`, and `accounting`
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If any file in "Current state" has drifted from the excerpts above.
- If a `pgx.Batch` loop fails to close `br.Close()` before issuing the next query on the same transaction connection (`pgx` requires `br.Close()` before the connection can execute another query, such as `UPDATE inspections SET updated_at = NOW()` at `inspection_repo.go:185`). Always call `if err := br.Close(); err != nil { return err }` immediately after draining `br.Exec()`!

## Maintenance notes

- Always close `pgx.BatchResults` (`br.Close()`) before running any subsequent SQL statement on the same `pgx.Tx` connection.
