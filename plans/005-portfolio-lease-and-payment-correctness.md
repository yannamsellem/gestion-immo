# Plan 005: Fix CSV whole-Euro parsing, rent call date/transaction bounds, shareholder share cap, and payment concurrency

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in `portfolio`, `lease`, and `payment` against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: `plans/003-http-limits-observability-and-headers.md`
- **Category**: bug | tests
- **Covers findings**: `BUG-01`, `BUG-05`, `BUG-06`, `BUG-07`, `TEST-02` (payment test suite)
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

Four high-impact domain bugs currently affect `portfolio`, `lease`, and `payment`:
1. In `csv_importer.go`, when a bank CSV uses an `amount`, `montant`, or `amount_eur` column and a row contains a whole Euro number without decimals (e.g. `"800"` or `"800 €"`), `parseAmountToCents` parses it as `800` cents (**8,00 €**) instead of **800,00 €** (`80000` cents), leaving 99% of the rent call unpaid (`BUG-01`).
2. `GenerateMonthlyRentCalls` ignores `lease.StartDate` and `lease.EndDate`, billing active leases for periods before they started or after they ended, and runs outside `UnitOfWork` (`BUG-05`).
3. Concurrent payments on the same lease read `ListUnpaidRentCallsByLease` without `FOR UPDATE` (lost-update race), and `processBankRows` commits each CSV row in a separate transaction without checking `external_ref` for idempotency (`BUG-06`).
4. `UpsertShareholder` checks `cmd.SharesCount <= entity.TotalShares` for a single shareholder in isolation, allowing multiple shareholders on the same entity to exceed 100% of `entity.TotalShares` (`BUG-07`).
Finally, `payment/application` currently has zero `*_test.go` files (`TEST-02`).

## Current state

- `api/internal/modules/payment/infrastructure/csv_importer.go:138-150, 225-237`:
  ```go
  		if centsStr, ok := getCol("amount_cents", "montant_cents"); ok && centsStr != "" {
  			parsed, err := strconv.ParseInt(centsStr, 10, 64)
  			...
  			amountCents = money.AmountCents(parsed)
  		} else if amtStr, ok := getCol("amount", "montant", "amount_eur"); ok && amtStr != "" {
  			parsed, err := parseAmountToCents(amtStr)
  			...
  			amountCents = parsed
  		}
  ...
  func parseAmountToCents(raw string) (money.AmountCents, error) {
  	cleaned := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
  	cleaned = strings.TrimSuffix(cleaned, "€")
  	cleaned = strings.TrimSpace(cleaned)
  	if strings.Contains(cleaned, ".") || strings.Contains(cleaned, ",") {
  		return parseDecimalEURToCents(cleaned)
  	}
  	val, err := strconv.ParseInt(cleaned, 10, 64)
  	if err != nil {
  		return 0, fmt.Errorf("%w: invalid amount %q", domain.ErrValidation, raw)
  	}
  	return money.AmountCents(val), nil
  }
  ```
- `api/internal/modules/lease/application/service.go:333-345` (`GenerateMonthlyRentCalls`):
  ```go
  	for _, lease := range leases {
  		if !lease.Active {
  			continue
  		}

  		existing, err := s.rentCallRepo.FindRentCallByLeaseAndPeriod(ctx, lease.ID, trimmedPeriod)
  ```
- `api/internal/modules/lease/infrastructure/postgres/rent_call_repo.go:146-155` (`ListUnpaidRentCallsByLease`):
  ```go
  func (r *RentCallRepo) ListUnpaidRentCallsByLease(ctx context.Context, leaseID string) ([]domain.RentCall, error) {
  	const query = `
  		SELECT id, lease_id, legal_entity_id, tenant_id, period_month,
  		       rent_net_cents, vat_cents, charges_cents, total_due_cents, remaining_due_cents,
  		       status, due_date, created_at, updated_at
  		FROM rent_calls
  		WHERE lease_id = $1 AND remaining_due_cents > 0 AND status <> 'PAID'
  		ORDER BY due_date ASC, period_month ASC
  	`
  ```
- `api/internal/modules/portfolio/application/service.go:227-229` (`UpsertShareholder`):
  ```go
  	if cmd.SharesCount < 0 || cmd.SharesCount > entity.TotalShares {
  		return ShareholderDTO{}, fmt.Errorf("%w: shares_count (%d) must be between 0 and total_shares (%d)", domain.ErrValidation, cmd.SharesCount, entity.TotalShares)
  	}
  ```

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Portfolio/Lease/Payment tests | `cd api && go test -v ./internal/modules/portfolio/... ./internal/modules/lease/... ./internal/modules/payment/...` | exit 0 |

## Scope

**In scope**:

- `api/internal/modules/portfolio/application/service.go`
- `api/internal/modules/portfolio/application/service_test.go`
- `api/internal/modules/portfolio/infrastructure/memory/shareholder_repo.go`
- `api/internal/modules/lease/application/service.go`
- `api/internal/modules/lease/application/service_test.go`
- `api/internal/modules/lease/infrastructure/memory/rent_call_repo.go`
- `api/internal/modules/lease/infrastructure/postgres/rent_call_repo.go`
- `api/internal/modules/payment/infrastructure/csv_importer.go`
- `api/internal/modules/payment/application/service.go`
- `api/internal/modules/payment/application/service_test.go` (create)

**Out of scope**:

- `auth`, `document`, `inspection`, and `accounting` modules.

## Git workflow

- Branch: `advisor/005-portfolio-lease-and-payment-correctness`
- Commit message style: `fix(payment,lease,portfolio): fix csv euro parsing, rent call bounds, share cap, and concurrency`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Fix `UpsertShareholder` aggregate share cap and case-sensitivity in `portfolio` (`BUG-07`)

1. In `api/internal/modules/portfolio/infrastructure/memory/shareholder_repo.go:31`, change `if strings.EqualFold(existing.FullName, shareholder.FullName)` to `if existing.FullName == shareholder.FullName` to match PostgreSQL's `ON CONFLICT (legal_entity_id, full_name)`.
2. In `api/internal/modules/portfolio/application/service.go` inside `UpsertShareholder` (lines 222–258):
   - Wrap the check-and-upsert inside `s.uow.WithinTransaction`.
   - Load `existingList, err := repos.Shareholders.ListShareholdersByEntity(txCtx, entity.ID)`.
   - Sum `sh.SharesCount` for all existing shareholders where `sh.FullName != fullName`. If `otherSharesSum + cmd.SharesCount > entity.TotalShares`, return `fmt.Errorf("%w: total shares across shareholders (%d) would exceed entity total_shares (%d)", domain.ErrValidation, otherSharesSum+cmd.SharesCount, entity.TotalShares)`.

**Verify**: `cd api && go test -v ./internal/modules/portfolio/...` → exit 0.

### Step 2: Enforce `StartDate`/`EndDate` bounds and `UnitOfWork` in `GenerateMonthlyRentCalls` (`BUG-05`)

1. In `api/internal/modules/lease/infrastructure/memory/rent_call_repo.go:26-37` (`SaveRentCalls`), check if another rent call with a *different* `ID` already exists for the same `(rc.LeaseID, rc.PeriodMonth)`; if so, return `domain.ErrDuplicatePeriod` (matching PostgreSQL's `UNIQUE (lease_id, period_month)`).
2. In `api/internal/modules/lease/application/service.go:333-381` (`GenerateMonthlyRentCalls`):
   - For each active lease, check `startMonth := lease.StartDate.UTC().Format("2006-01")`; if `trimmedPeriod < startMonth`, skip (`continue`).
   - If `lease.EndDate != nil`, check `endMonth := lease.EndDate.UTC().Format("2006-01")`; if `trimmedPeriod > endMonth`, skip (`continue`).
   - Execute the `FindRentCallByLeaseAndPeriod` loop and `SaveRentCalls` inside `s.uow.WithinTransaction`.

**Verify**: `cd api && go test -v ./internal/modules/lease/...` → exit 0.

### Step 3: Fix CSV whole-Euro parsing, `FOR UPDATE` locking, and transactional/idempotent `processBankRows` (`BUG-01`, `BUG-06`)

1. In `api/internal/modules/payment/infrastructure/csv_importer.go`:
   - When a header CSV uses `amount`, `montant`, or `amount_eur` (lines 144–150), call `parseDecimalEURToCents(strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(amtStr, " ", ""), "€")))` directly instead of `parseAmountToCents(amtStr)`. Because `parseDecimalEURToCents` splits on `.` and multiplies `parts[0]` by `100` even when `len(parts) == 1` (lines 254–260), `"800"` and `"800 €"` under `amount`/`montant`/`amount_eur` will properly parse to `80000` cents (`800,00 €`), while `amount_cents`/`montant_cents` (lines 138–143) continues to parse raw integer cents!
   - In the positional fallback (line 157, where the column is documented as `amount_cents (or EUR)`), keep `parseAmountToCents(rec[1])` so existing positional `amount_cents` tests continue to pass, unless `rec[1]` contains `"€"`, `"."`, or `","`, in which case `parseAmountToCents` should call `parseDecimalEURToCents`.
2. In `api/internal/modules/lease/infrastructure/postgres/rent_call_repo.go:146-155` (`ListUnpaidRentCallsByLease`):
   - If `postgres.TxFromContext(ctx) != nil`, append ` FOR UPDATE` to the `SELECT ... FROM rent_calls WHERE lease_id = $1 ... ORDER BY due_date ASC, period_month ASC` query so concurrent payment allocations on the same lease serialize cleanly.
3. In `api/internal/modules/payment/application/service.go:468-533` (`processBankRows`):
   - Pre-validate and resolve `leaseID` for all rows before mutating any state (so a row with an unresolvable `lease_id` fails before any row is persisted).
   - Load existing payments for the entity (`s.paymentRepo.ListPaymentsByEntity(ctx, entityID)`) and index non-empty `ExternalRef` values; if an incoming row has a non-empty `ExternalRef` that was already imported (or duplicated within the same batch), skip it idempotently.
   - Wrap the batch execution in `s.uow.WithinTransaction`.

**Verify**: `cd api && go test -v ./internal/modules/payment/...` → exit 0.

### Step 4: Create `api/internal/modules/payment/application/service_test.go` (`TEST-02`)

1. Create `api/internal/modules/payment/application/service_test.go` (modeled after `api/internal/modules/lease/application/service_test.go`) covering:
   - `ImportPaymentsCSV` with a header `lease_id,amount_eur,label,payment_date,external_ref` containing whole-Euro `"800"` and `"800 €"` -> asserts `80000` cents (`800,00 €`) is allocated and re-importing the same `external_ref` is idempotent.
   - `ImportPaymentsCSV` atomic rollback when a later row has an unresolvable `lease_id`.
   - `RecordCashPayment` and `RecordCashDepositToBank` (`530` -> `580` -> `512` journal entries) and `ListCashJournalEntries`.
   - `ParseOFX` bank statement import.
   - All 6 HTTP endpoints (`POST /api/v1/payments/cash`, `/bank`, `/cash-deposit`, `/import-csv`, `GET /api/v1/entities/{id}/negative-positions`, `/cash-journal`) and all 3 MCP tools (`record_cash_payment`, `record_bank_payment`, `list_negative_positions`).

**Verify**: `cd api && go vet ./... && go test -v ./internal/modules/payment/...` → exit 0, all new tests pass.

## Test plan

- Add `TestUpsertShareholder_RejectsExceedingTotalShares` in `api/internal/modules/portfolio/application/service_test.go`.
- Add `TestGenerateMonthlyRentCalls_RespectsLeaseStartAndEndDates` in `api/internal/modules/lease/application/service_test.go`.
- Create `api/internal/modules/payment/application/service_test.go` with the 5 test cases listed in Step 4.
- Run `cd api && go test -race ./internal/modules/portfolio/... ./internal/modules/lease/... ./internal/modules/payment/...` → all pass.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test -race ./...` exits 0
- [ ] `api/internal/modules/payment/application/service_test.go` exists and passes
- [ ] Whole-Euro `"800"` under `amount`/`montant`/`amount_eur` CSV header parses as `80000` cents (`800,00 €`)
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If any existing test in `api/internal/modules/payment/domain/allocation_service_test.go` fails because it passes headerless positional CSVs with raw cents, verify Step 3.1 kept `parseAmountToCents` for positional rows without `"€"`, `"."`, or `","`.
- If any file in "Current state" has drifted from the excerpts above.

## Maintenance notes

- When Plan 009 wires automatic double-entry accounting (`DIR-01`) into `payment.Service`, the new tests in `api/internal/modules/payment/application/service_test.go` will serve as the regression suite.
