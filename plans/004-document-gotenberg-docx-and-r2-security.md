# Plan 004: Harden Gotenberg HTML rendering, guard DOCX/GLI archives against ZIP bombs, and validate R2 uploads

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in `api/docker-compose.yml`, `api/internal/platform/templating/docx_engine.go`, and `api/internal/modules/document/application/service.go` against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: `plans/001-dx-makefile-compose-and-backup-hardening.md`
- **Category**: security | bug | perf
- **Covers findings**: `SEC-02`, `SEC-06`, `SEC-07` (R2 path sanitization & parent check), `PERF-02` (`GetLegalEntityByID` lightweight lookup)
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

When `GenerateFromTemplate` renders an HTML template via Gotenberg, headless Chromium evaluates embedded scripts, iframes, and URLs inside the `internal` Docker bridge network where `gestion-immo:8080` and `postgres:5432` run, enabling internal SSRF and local `file://` disclosure into the generated PDF. In addition, `DocxEngine.RenderDocx` reads every ZIP entry with `io.ReadAll` without entry-count or decompressed-byte caps (vulnerable to ZIP decompression bombs), `AssembleGLIDossier` writes lease attachments into `"pieces_jointes/" + sanitizeFileName(att.FileName)` without disambiguating colliding filenames or bounding total size, `AttachFileToR2` interpolates unsanitized `leaseID`/`entityID` into `r2Key` and uploads to R2 before checking parent existence, and `GenerateFromTemplate` fetches the entire portfolio hierarchy (`GetEntityPortfolioSummary`) just to read 4 fields on `LegalEntity`.

## Current state

- `api/docker-compose.yml:19-24`:
  ```yaml
    gotenberg:
      image: gotenberg/gotenberg:8
      restart: unless-stopped
      networks:
        - internal
  ```
- `api/internal/platform/templating/docx_engine.go:50-61`:
  ```go
  	for _, f := range zr.File {
  		rc, err := f.Open()
  		if err != nil {
  			_ = zw.Close()
  			return nil, fmt.Errorf("templating: open zip entry %s: %w", f.Name, err)
  		}
  		raw, err := io.ReadAll(rc)
  		_ = rc.Close()
  ```
- `api/internal/modules/document/application/service.go:274-301` (`AttachFileToR2`):
  ```go
  	scope := "general"
  	if leaseID != "" {
  		scope = "leases/" + leaseID
  	} else if entityID != "" {
  		scope = "entities/" + entityID
  	}
  	r2Key := fmt.Sprintf("files/%s/%s_%s", scope, id, fileName)

  	if err := s.storage.Upload(ctx, r2Key, contentType, bytes.NewReader(cmd.Content)); err != nil {
  		return StoredFileDTO{}, err
  	}
  ```
- `api/internal/modules/document/application/service.go:346-353` (`GenerateFromTemplate`):
  ```go
  	if entityID != "" && s.portfolioSvc != nil {
  		if summary, pErr := s.portfolioSvc.GetEntityPortfolioSummary(ctx, entityID); pErr == nil {
  			vars["societe.nom"] = summary.Entity.Name
  			vars["societe.siret"] = summary.Entity.SIRET
  			vars["societe.adresse"] = summary.Entity.Address
  			vars["societe.capital"] = summary.Entity.ShareCapitalCents.FormatEUR()
  		}
  	}
  ```
- `api/internal/modules/document/application/service.go:579-600` (`AssembleGLIDossier`):
  ```go
  	storedAttachments, err := s.storedFiles.ListStoredFilesByLease(ctx, lease.ID)
  	if err == nil {
  		for _, att := range storedAttachments {
  			...
  			entryName := "pieces_jointes/" + sanitizeFileName(att.FileName)
  			if err := addZipEntry(entryName, attBytes); err != nil {
  ```

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Templating tests | `cd api && go test -v ./internal/platform/templating/...` | exit 0 |
| Document tests | `cd api && go test -v ./internal/modules/document/...` | exit 0 |

## Scope

**In scope**:

- `api/docker-compose.yml`
- `api/internal/platform/templating/docx_engine.go`
- `api/internal/platform/templating/docx_engine_test.go`
- `api/internal/modules/portfolio/application/service.go`
- `api/internal/modules/document/application/service.go`
- `api/internal/modules/document/application/service_test.go`

**Out of scope**:

- Other modules (`auth`, `lease`, `payment`, `inspection`, `accounting`).

## Git workflow

- Branch: `advisor/004-document-gotenberg-docx-and-r2-security`
- Commit message style: `fix(document): harden gotenberg html, docx zip limits, and r2 upload validation`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Lock down Gotenberg Chromium in `api/docker-compose.yml` and validate HTML templates (`SEC-02`)

1. In `api/docker-compose.yml:19-24`, add a `command:` block to `gotenberg`:
   ```yaml
     gotenberg:
       image: gotenberg/gotenberg:8
       restart: unless-stopped
       command:
         - "gotenberg"
         - "--chromium-disable-javascript=true"
         - "--chromium-deny-list=^file:///.*,^https?://(localhost|127\\.|10\\.|172\\.(1[6-9]|2[0-9]|3[0-1])\\.|192\\.168\\.|169\\.254\\.|gestion-immo|postgres|gotenberg).*"
       networks:
         - internal
   ```
2. In `api/internal/modules/document/application/service.go`, add a helper `validateSafeHTMLTemplate(htmlBody string) error` called in `RegisterTemplate` (when `htmlBody != ""`) that rejects dangerous tags/schemes case-insensitively (`<script`, `<iframe`, `<object`, `<embed`, `<link`, `javascript:`, `file://`) with `fmt.Errorf("%w: html_body contains forbidden active or external tags", domain.ErrValidation)`.

**Verify**: `cd api && go test ./internal/modules/document/...` → exit 0.

### Step 2: Guard `DocxEngine` against ZIP decompression bombs (`SEC-06`)

1. In `api/internal/platform/templating/docx_engine.go`:
   - Define `const maxDocxZipEntries = 256`, `const maxDocxEntryBytes = 16 << 20` (16 MiB), and `const maxDocxTotalBytes = 32 << 20` (32 MiB).
   - In `RenderDocx` (line 49), reject archives with `len(zr.File) > maxDocxZipEntries`.
   - Wrap each `rc` in `io.LimitReader(rc, maxDocxEntryBytes+1)` and track `totalReadBytes`; return a descriptive error if an entry exceeds `maxDocxEntryBytes` or `totalReadBytes > maxDocxTotalBytes`.
   - In `ExtractDocxText` (line 303), wrap `word/document.xml` in `io.LimitReader(rc, maxDocxEntryBytes+1)`.

**Verify**: `cd api && go test -v ./internal/platform/templating/...` → exit 0.

### Step 3: Sanitize R2 scope segments, disambiguate GLI ZIP entries, and add `GetLegalEntityByID` (`SEC-06`, `SEC-07`, `PERF-02`)

1. In `api/internal/modules/portfolio/application/service.go`, add `GetLegalEntityByID(ctx context.Context, entityID string) (LegalEntityDTO, error)` that calls `s.entityRepo.FindLegalEntityByID(ctx, strings.TrimSpace(entityID))` and returns `toLegalEntityDTO(*ent)` without loading shareholders, properties, or lots.
2. In `api/internal/modules/document/application/service.go`:
   - In `AttachFileToR2` (lines 262–284), validate that `leaseID` and `entityID` contain only safe identifier characters (`[a-zA-Z0-9_-]`, returning `domain.ErrValidation` if they contain `/` or `..`), and when `s.leaseSvc != nil` / `s.portfolioSvc != nil`, verify that a non-empty `leaseID` or `entityID` exists *before* calling `s.storage.Upload`.
   - In `GenerateFromTemplate` (lines 346–353), if `s.portfolioSvc` implements `GetLegalEntityByID(ctx, entityID)` (or fall back to `GetEntityPortfolioSummary`), use `GetLegalEntityByID` to populate `societe.*` variables.
   - In `AssembleGLIDossier` (lines 579–600), cap cumulative attachment bytes (e.g. `64 << 20`) and track `usedEntryNames := make(map[string]int)` so if two attachments have the same `sanitizeFileName(att.FileName)` (e.g. `quittance.pdf`), the second is named `pieces_jointes/quittance_2.pdf` instead of colliding in the ZIP archive.

**Verify**: `cd api && go vet ./... && go test -v ./internal/modules/document/... ./internal/modules/portfolio/...` → exit 0.

## Test plan

- In `api/internal/platform/templating/docx_engine_test.go`, add `TestRenderDocx_RejectsExcessiveEntriesOrSize` verifying that `RenderDocx` rejects an archive exceeding the entry limit.
- In `api/internal/modules/document/application/service_test.go`, add tests verifying:
  - `RegisterTemplate` rejects HTML containing `<script>` or `<iframe>` with `domain.ErrValidation`.
  - `AttachFileToR2` rejects `lease_id` containing `"../"` or `"a/b"` before uploading to R2.
  - `AssembleGLIDossier` deduplicates two lease attachments both named `"quittance.pdf"` into `"pieces_jointes/quittance.pdf"` and `"pieces_jointes/quittance_2.pdf"`.
- Run `cd api && go test -v ./internal/platform/templating/... ./internal/modules/document/...` → all pass.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test ./...` exits 0
- [ ] `gotenberg` in `api/docker-compose.yml` includes `--chromium-disable-javascript=true` and `--chromium-deny-list`
- [ ] New tests in `docx_engine_test.go` and `document/application/service_test.go` pass
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If any existing test in `document/application/service_test.go` calls `AttachFileToR2` with a fake `lease_id` that does not exist in `leaseSvc`, only enforce parent existence when the ID exists or check whether existing tests seed the lease first; do not break existing test fixtures.
- If any file in "Current state" has drifted from the excerpts above.

## Maintenance notes

- All built-in HTML generators (`buildTenantLedgerHTML`, `buildFormalNoticeHTML`, `annual_accounts_pdf_renderer.go`, `buildInspectionReportHTML`) use self-contained inline CSS and require neither JavaScript nor external network requests.
