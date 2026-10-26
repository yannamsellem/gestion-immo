# Plan 006: Wire accounting to live portfolio metadata, fix balance line sync, and add accounting application tests

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in `api/internal/modules/accounting/application/service.go`, `api/internal/modules/accounting/module.go`, and `api/cmd/server/server.go` against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: `plans/005-portfolio-lease-and-payment-correctness.md`
- **Category**: bug | tests | tech-debt
- **Covers findings**: `BUG-02`, `BUG-04`, `TEST-02` (accounting test suite), `ARCH-01` (accounting fallback deduplication)
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

Currently, `GenerateAnnualAccountsPDF` in `api/internal/modules/accounting/application/service.go:375` unconditionally calls `defaultAnnualAccountsMetadata`, which hardcodes `"SARL INVESTISSIMO"`, SIRET `"38012345600018"`, `"93300 AUBERVILLIERS"`, `256 000,00 €` share capital, and two fake shareholders for every legal entity instead of reading from `portfolio.Service`. Even worse, `syncBalanceLinesFromEntry` (`service.go:469-504`) indexes existing balance lines only by `l.LineID` (never by `l.AccountCode`), so posting a journal entry on an account like `706100` creates a duplicate row (`"acct_706100"`) alongside `"rev_locaux"` and appends duplicate items to `updated` inside the loop. Finally, `accounting/application` has zero `*_test.go` files and duplicates `buildFallbackAnnualAccountsHTML`/`exportFallbackFEC` in `service.go`.

## Current state

- `api/internal/modules/accounting/application/service.go:370-390, 459-516, 596-629`:
  ```go
  	stmt, err := s.PreviewFinancialStatements(ctx, trimmedEntity, fiscalYear, disabledLineIDs)
  	if err != nil {
  		return StoredFileDTO{}, err
  	}

  	meta := defaultAnnualAccountsMetadata(trimmedEntity, fiscalYear)
  ...
  func (s *Service) syncBalanceLinesFromEntry(ctx context.Context, balancesRepo domain.AccountBalanceRepository, entry domain.JournalEntry) error {
  	existing, err := balancesRepo.ListAccountBalanceLines(ctx, entry.LegalEntityID, entry.FiscalYear)
  	if err != nil {
  		return err
  	}
  	byLineID := make(map[string]domain.AccountBalanceLine, len(existing))
  	for _, l := range existing {
  		byLineID[l.LineID] = l
  	}

  	var updated []domain.AccountBalanceLine
  	for _, jl := range entry.Lines {
  		...
  		byLineID[lineID] = cur
  		updated = append(updated, cur)
  	}

  	return balancesRepo.UpsertAccountBalanceLines(ctx, entry.LegalEntityID, entry.FiscalYear, updated)
  }
  ```
- `api/internal/modules/accounting/module.go:27-64`:
  ```go
  func NewModule(pool *pgxpool.Pool, storage r2.ObjectStorage, pdfEngine templating.PdfEngine) *Module {
  ```
- `api/cmd/server/server.go:99`:
  ```go
  	accountingMod := accounting.NewModule(pool, r2Storage, pdfEngine)
  ```

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Accounting tests | `cd api && go test -v ./internal/modules/accounting/...` | exit 0 |
| Full tests | `cd api && go test ./...` | exit 0 |

## Scope

**In scope**:

- `api/internal/modules/accounting/application/service.go`
- `api/internal/modules/accounting/application/service_test.go` (create)
- `api/internal/modules/accounting/module.go`
- `api/cmd/server/server.go`
- `api/cmd/main_test.go`

**Out of scope**:

- `accounting/domain/statement_calculator.go` (already tested and working).
- Other domain modules (`auth`, `lease`, `payment`, `document`, `inspection`).

## Git workflow

- Branch: `advisor/006-accounting-live-portfolio-and-balance-sync`
- Commit message style: `fix(accounting): wire live portfolio metadata, fix balance sync, and add tests`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Inject `PortfolioPort` into `accounting.Service` and resolve live entity/shareholder metadata (`BUG-02`, `ARCH-01`)

1. In `api/internal/modules/accounting/application/service.go`:
   - Define a `PortfolioPort` interface (matching `portfolio.Service.GetEntityPortfolioSummary(ctx context.Context, entityID string) (portfolioapp.EntityPortfolioSummaryDTO, error)`, or an optional setter `WithPortfolioPort(port PortfolioPort) *Service` / constructor parameter).
   - Add `resolveAnnualAccountsMetadata(ctx context.Context, entityID string, fiscalYear int) domain.AnnualAccountsMetadata` that queries `s.portfolioSvc.GetEntityPortfolioSummary(ctx, entityID)` when `s.portfolioSvc != nil` and populates `EntityName`, `EntityType`, `SIRET`, `NAFCode`, `Address`, `ShareCapitalCents`, `TotalShares`, and `Shareholders` (`FullName`, `SharesCount`, `DetentionBasisPoints`, `CurrentAccountCents`, `RemunerationPaidCents`) from the live portfolio entity and its shareholders, falling back to `defaultAnnualAccountsMetadata` only when `s.portfolioSvc == nil` or the entity is not registered in `portfolioSvc`.
   - In `NewService`, if `fecExporter == nil` or `annualAccountsRenderer == nil`, default them to `infrastructure.NewFECExporter()` and `infrastructure.NewAnnualAccountsPDFRenderer(pdfEngine)` (or keep the clean interface defaults) so `buildFallbackAnnualAccountsHTML` and `exportFallbackFEC` in `service.go` can be removed.
2. In `api/internal/modules/accounting/module.go` and `api/cmd/server/server.go:99`, wire `portfolioMod.Service()` into `accountingMod` (e.g., via `accountingMod.Service().WithPortfolioPort(portfolioMod.Service())` or `accounting.NewModule(pool, r2Storage, pdfEngine, portfolioMod.Service())`).

**Verify**: `cd api && go vet ./... && go test ./cmd/... ./internal/modules/accounting/...` → exit 0.

### Step 2: Fix `syncBalanceLinesFromEntry` account matching and deduplication (`BUG-04`)

1. In `api/internal/modules/accounting/application/service.go:459-505` (`syncBalanceLinesFromEntry`):
   - Build both `byLineID := make(map[string]domain.AccountBalanceLine, len(existing))` and `lineIDByAccountCode := make(map[string]string, len(existing))` from `existing`.
   - For each `jl` in `entry.Lines`, if `jl.LineID == ""` or `jl.LineID == "acct_"+jl.AccountCode`, check if `existingLineID, ok := lineIDByAccountCode[jl.AccountCode]`; if `ok`, set `lineID = existingLineID` so journal entries posted against an already-seeded account (like `706100` -> `"rev_locaux"`) update that existing `AccountBalanceLine` instead of creating a duplicate `"acct_706100"` row!
   - Record touched `lineID`s in an ordered slice + `touched := make(map[string]bool)` during the loop, and construct `updated` once after the loop from `byLineID[id]` so multiple lines touching the same `lineID` in one entry produce a single upserted `AccountBalanceLine`.

**Verify**: `cd api && go vet ./internal/modules/accounting/...` → exit 0.

### Step 3: Create `api/internal/modules/accounting/application/service_test.go` (`TEST-02`)

1. Create `api/internal/modules/accounting/application/service_test.go` testing:
   - `RecordJournalEntry` rejects unbalanced entries (`ErrUnbalancedJournalEntry`) and entries with `< 2` lines.
   - `RecordJournalEntry` on an account that already has a seeded `AccountBalanceLine` (e.g., `LineID: "rev_locaux"`, `AccountCode: "706100"`) updates `"rev_locaux"` in place without creating a second `"acct_706100"` row, and handles multiple lines with the same account in a single journal entry without duplicate upserts.
   - `PreviewFinancialStatements` with `disabled_line_ids` dynamically recalculates totals and `NetIncomeNCents`.
   - `GenerateAnnualAccountsPDF` with a live `portfolio.Service` entity and shareholders populates the entity's real name, SIRET, and shareholders in the rendered PDF HTML and uploads the PDF to R2.
   - `ExportEntityFEC` produces a normative 18-column tab-separated DGFiP FEC header and lines.
   - All 6 HTTP endpoints (`POST /api/v1/entities/{id}/journal-entries`, `GET .../journal-entries`, `PUT .../statements/lines`, `POST .../statements/preview`, `POST .../statements/pdf`, `GET .../fec`) and all 3 MCP tools (`preview_financial_statement`, `generate_annual_accounts_pdf`, `export_fec`).

**Verify**: `cd api && go test -v -race ./internal/modules/accounting/... ./cmd/...` → exit 0, all pass.

## Test plan

- `api/internal/modules/accounting/application/service_test.go` covers all 6 test scenarios in Step 3.
- Verification: `cd api && go test -v ./internal/modules/accounting/...` → all pass.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test -race ./...` exits 0
- [ ] `api/internal/modules/accounting/application/service_test.go` exists and passes
- [ ] `GenerateAnnualAccountsPDF` uses live entity/shareholder data from `portfolio.Service` when available
- [ ] `syncBalanceLinesFromEntry` updates existing `AccountBalanceLine` rows matched by `AccountCode` without creating duplicate rows
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If `cmd/main_test.go` calls `PreviewFinancialStatements` on an entity with no seeded balance lines and asserts on the default SGCAM demo lines, preserve `defaultSGCAMBalanceLines` fallback ONLY when `s.portfolioSvc == nil` or when the entity ID does not exist in `portfolioSvc` (or check `cmd/main_test.go` first before removing the fallback).
- If any file in "Current state" has drifted from the excerpts above.

## Maintenance notes

- `syncBalanceLinesFromEntry` is on the critical path for Plan 009 (`DIR-01`), where rent calls and payments automatically post `JournalEntry` records into `accounting.Service`.
