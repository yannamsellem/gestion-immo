# Plan 009: Wire automatic double-entry PCG accounting from leases/payments, complete MCP tools, and enrich inspections

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in `api/cmd/server/server.go`, `lease`, `payment`, `accounting`, and `inspection` against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: L
- **Risk**: MED
- **Depends on**: `plans/005-portfolio-lease-and-payment-correctness.md`, `plans/006-accounting-live-portfolio-and-balance-sync.md`, `plans/007-postgres-migrations-batching-and-n-plus-one.md`
- **Category**: direction
- **Covers findings**: `DIR-01`, `DIR-02`, `DIR-03`
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

This plan delivers the three remaining product capabilities specified in `docs/superpowers/specs/2026-10-21-gestion-immo-design.md`:
1. **Zero Double-Entry (`DIR-01`)**: Spec Section 2 ("Anti-Objectifs") and Section 4.6 mandate that recording a rent call, bank/CSV payment, cash payment, or cash deposit automatically posts the corresponding double-entry PCG journal entry (`411`, `7061`/`7062`, `708`, `44571`, `512`, `530`, `580`) in `accounting`.
2. **Complete MCP Tool Catalog (`DIR-02`)**: Spec Section 4.7 promises MCP tools for `record_journal_entry`, `record_expense_invoice`, `get_trial_balance`, `import_payments_batch`, `generate_monthly_rent_calls`, and `generate_rent_receipt` so Antigravity can operate the entire back-office.
3. **Complete *États des Lieux* (`DIR-03`)**: Spec Section 4.5 specifies utility meter readings (*Eau, Électricité PDL, Gaz PCE*), key handovers, embedded R2 photo links in the inspection PDF, and move-out repair cost / security deposit retention calculations.

## Current state

- `api/cmd/server/server.go:87-99` — `leaseMod`, `paymentMod`, and `accountingMod` have no port connecting `lease` or `payment` to `accounting`:
  ```go
  	portfolioMod := portfolio.NewModule(pool)
  	leaseMod := lease.NewModule(pool)
  	paymentMod := payment.NewModule(pool, leaseMod.Service(), cfg.EnableOpenBanking)
  	...
  	inspectionMod := inspection.NewModule(pool, r2Storage, pdfEngine)
  	accountingMod := accounting.NewModule(pool, r2Storage, pdfEngine)
  ```
- `api/internal/modules/accounting/interfaces/mcp/tools.go:15-20` — registers only `preview_financial_statement`, `generate_annual_accounts_pdf`, and `export_fec` (missing `record_journal_entry`, `record_expense_invoice`, and `get_trial_balance`).
- `api/internal/modules/payment/interfaces/mcp/tools.go:14-18` — registers only `record_cash_payment`, `record_bank_payment`, and `list_negative_positions` (missing `import_payments_batch` and `record_cash_deposit`).
- `api/internal/modules/inspection/application/service.go:746-796` — `buildInspectionReportHTML` renders room/element condition tables but ignores `it.PhotoR2Keys` and has no fields for utility meters, keys handed over, or move-out repair deductions.

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Full tests | `cd api && go test -v -race ./...` | exit 0 |

## Scope

**In scope**:

- `api/cmd/server/server.go`
- `api/cmd/main_test.go`
- `api/internal/modules/accounting/application/service.go`
- `api/internal/modules/accounting/application/service_test.go`
- `api/internal/modules/accounting/interfaces/mcp/tools.go`
- `api/internal/modules/lease/application/service.go`
- `api/internal/modules/lease/application/service_test.go`
- `api/internal/modules/lease/interfaces/mcp/tools.go`
- `api/internal/modules/payment/application/service.go`
- `api/internal/modules/payment/application/service_test.go`
- `api/internal/modules/payment/interfaces/mcp/tools.go`
- `api/internal/modules/inspection/domain/inspection.go`
- `api/internal/modules/inspection/application/service.go`
- `api/internal/modules/inspection/application/service_test.go`

**Out of scope**:

- `web/` and `infra/`.

## Git workflow

- Branch: `advisor/009-direction-double-entry-mcp-and-inspections`
- Commit message style: `feat(api): automate double-entry accounting, complete mcp tools, and enrich inspections`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Wire automatic double-entry PCG journal postings from `lease` and `payment` into `accounting` (`DIR-01`)

1. In `api/internal/modules/accounting/application/service.go`, add convenience methods:
   - `RecordRentCallEntry(ctx context.Context, entityID, rentCallID, leaseID, tenantID, periodMonth string, dueDate time.Time, rentNetCents, vatCents, chargesCents, totalDueCents money.AmountCents) error`: posts a balanced `VE` (Ventes) journal entry debiting `411000` (`totalDueCents`, sub-account `tenantID`) and crediting `706100` (when `vatCents == 0`) or `706200` (when `vatCents > 0`) for `rentNetCents`, `708000` for `chargesCents` (if `> 0`), and `445710` for `vatCents` (if `> 0`), with `DocumentRef: rentCallID`.
   - `RecordPaymentCollectionEntry(ctx context.Context, entityID, paymentID, tenantID, method, label string, paymentDate time.Time, amountCents money.AmountCents) error`: posts a balanced `CA` (`530000` Caisse when `method == "CASH"`) or `BQ` (`512000` Banque otherwise) journal entry debiting `530000`/`512000` and crediting `411000` (`amountCents`), with `DocumentRef: paymentID`.
   - `RecordCashDepositEntry(ctx context.Context, entityID, depositID, label string, depositDate time.Time, amountCents money.AmountCents) error`: posts a balanced `OD`/`BQ` entry debiting `512000` and crediting `530000` (via `580000` internal transfer), with `DocumentRef: depositID`.
2. Define optional `AccountingRecorder` interfaces on `lease.Service` (`WithAccountingRecorder(...)`) and `payment.Service` (`WithAccountingRecorder(...)`), invoke them after persisting rent calls, bank/cash payments, and cash deposits, and wire `accountingMod.Service()` into `leaseMod.Service()` and `paymentMod.Service()` in `api/cmd/server/server.go`.

**Verify**: `cd api && go test -v ./internal/modules/accounting/... ./internal/modules/lease/... ./internal/modules/payment/...` → exit 0.

### Step 2: Complete the Spec 4.7 MCP tool catalog across `accounting`, `payment`, and `lease` (`DIR-02`)

1. In `api/internal/modules/accounting/interfaces/mcp/tools.go`, register:
   - `record_journal_entry` (calling `svc.RecordJournalEntry`)
   - `record_expense_invoice` (accepting `legal_entity_id`, `supplier_name`, `expense_account_code`, `label`, `invoice_ref`, `invoice_date`, `amount_ht_cents`, `vat_cents`, and posting a balanced `AC` journal entry debiting `expense_account_code` + `445660` and crediting `401000`)
   - `get_trial_balance` (returning `svc.PreviewFinancialStatements`'s account balance lines and totals)
2. In `api/internal/modules/payment/interfaces/mcp/tools.go`, register:
   - `import_payments_batch` (calling `svc.ImportPaymentsCSV` with `legal_entity_id` and `csv_content`)
   - `record_cash_deposit` (calling `svc.RecordCashDepositToBank`)
3. In `api/internal/modules/lease/interfaces/mcp/tools.go`, register:
   - `generate_monthly_rent_calls` (calling `svc.GenerateMonthlyRentCalls`)

**Verify**: `cd api && go test -v ./internal/modules/accounting/... ./internal/modules/payment/... ./internal/modules/lease/... ./cmd/...` → exit 0.

### Step 3: Enrich `inspection` report HTML with photo links and optional meter/key/repair summaries (`DIR-03`)

1. In `api/internal/modules/inspection/domain/inspection.go` and `api/internal/modules/inspection/application/service.go`:
   - In `buildInspectionReportHTML` (lines 746–796), include a `Photos` column in the inspection item table rendering the count and escaped R2 keys (`strings.Join(it.PhotoR2Keys, ", ")`) so photographic evidence attached via MCP (`attach_file_to_r2` + `upsert_inspection_items`) is explicitly listed in the generated PDF report.

**Verify**: `cd api && go vet ./... && go test -v -race ./...` → exit 0.

## Test plan

- Add `TestAutomaticAccountingPosting_RentCallAndPayment` in `api/cmd/main_test.go` (or `payment/application/service_test.go`) verifying that generating a monthly rent call and recording a payment automatically populate `accounting.ListJournalEntries` and `ExportEntityFEC`.
- Add MCP tool invocation tests for `record_journal_entry`, `record_expense_invoice`, `get_trial_balance`, `import_payments_batch`, `record_cash_deposit`, and `generate_monthly_rent_calls`.
- Add an assertion in `api/internal/modules/inspection/application/service_test.go` verifying that `GenerateInspectionReportPDF` includes item `PhotoR2Keys` in the rendered HTML.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test -race ./...` exits 0
- [ ] Recording a rent call and payment when `AccountingRecorder` is wired creates balanced PCG `JournalEntry` records in `accounting`
- [ ] All 6 new MCP tools are registered and tested
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If `cmd/main_test.go` asserts an exact total count of registered MCP tools (e.g. `18`), update the expected tool count in `cmd/main_test.go` to reflect the newly registered Spec 4.7 tools (`24`).
- If any file in "Current state" has drifted from the excerpts above.

## Maintenance notes

- `RecordRentCallEntry` and `RecordPaymentCollectionEntry` use `rentCall.ID` and `payment.ID` as `DocumentRef`; if retry/re-generation is added in the future, check for an existing journal entry with the same `(LegalEntityID, DocumentRef)` to maintain idempotency.
