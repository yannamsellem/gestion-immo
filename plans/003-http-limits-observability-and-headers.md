# Plan 003: Bound HTTP request bodies, add structured error/request logging, and set security headers

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: Compare the "Current state" excerpts in `api/internal/platform/httpx/response.go`, `api/cmd/server/server.go`, and the module `interfaces/http/handler.go` files against live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: `plans/002-auth-route-protection-and-security-hardening.md`
- **Category**: security | dx
- **Covers findings**: `SEC-04`, `SEC-07` (HTTP headers & FEC `Content-Disposition`), `DX-03`
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

Currently, `httpx.DecodeJSON` (`api/internal/platform/httpx/response.go:100-111`), `handleImportPaymentsCSV` (`api/internal/modules/payment/interfaces/http/handler.go:95-151`), and base64 upload routes in `document` read arbitrary-sized request bodies into memory without `http.MaxBytesReader` or `io.LimitReader`, and `http.Server` in `api/cmd/server/server.go:129-133` omits `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`. Furthermore, all 7 HTTP handlers silently discard unhandled errors in the `default:` branch of `writeDomainError` when returning `500 Internal Server Error`, `handleExportFEC` interpolates unsanitized `{id}` into `Content-Disposition`, and responses lack `X-Content-Type-Options: nosniff` and `Cache-Control: no-store` on token endpoints.

## Current state

- `api/internal/platform/httpx/response.go:15-23, 95-111`:
  ```go
  func JSON(w http.ResponseWriter, status int, payload any) {
  	w.Header().Set("Content-Type", "application/json; charset=utf-8")
  	w.WriteHeader(status)
  	if payload == nil || status == http.StatusNoContent {
  		return
  	}
  	_ = json.NewEncoder(w).Encode(payload)
  }
  ...
  func InternalError(w http.ResponseWriter, message string) {
  	Error(w, http.StatusInternalServerError, message)
  }

  func DecodeJSON(r *http.Request, dst any) error {
  	if r.Body == nil {
  		return fmt.Errorf("request body is empty")
  	}
  	defer r.Body.Close()
  	dec := json.NewDecoder(r.Body)
  	if err := dec.Decode(dst); err != nil {
  		return fmt.Errorf("invalid JSON payload: %w", err)
  	}
  	return nil
  }
  ```
- `api/internal/modules/portfolio/interfaces/http/handler.go:138-151` (and identical `default:` blocks in `auth`, `lease`, `payment`, `document`, `inspection`, `accounting` HTTP handlers):
  ```go
  func writeDomainError(w http.ResponseWriter, err error) {
  	switch {
  	...
  	default:
  		httpx.InternalError(w, "internal portfolio error")
  	}
  }
  ```
- `api/internal/modules/accounting/interfaces/http/handler.go:146-148`:
  ```go
  	w.Header().Set("Content-Type", "text/tab-separated-values; charset=utf-8")
  	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%sFEC%d1231.txt\"", strings.TrimSpace(entityID), fiscalYear))
  ```
- `api/cmd/server/server.go:128-133`:
  ```go
  	srv := &http.Server{
  		Addr:              addr,
  		Handler:           handler,
  		ReadHeaderTimeout: 10 * time.Second,
  	}
  ```

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0 |
| Test | `cd api && go test ./...` | exit 0 |

## Scope

**In scope**:

- `api/internal/platform/httpx/response.go`
- `api/internal/platform/httpx/response_test.go` (create)
- `api/cmd/main.go`
- `api/cmd/server/server.go`
- `api/internal/modules/auth/interfaces/http/handler.go`
- `api/internal/modules/portfolio/interfaces/http/handler.go`
- `api/internal/modules/lease/interfaces/http/handler.go`
- `api/internal/modules/payment/interfaces/http/handler.go`
- `api/internal/modules/document/interfaces/http/handler.go`
- `api/internal/modules/inspection/interfaces/http/handler.go`
- `api/internal/modules/accounting/interfaces/http/handler.go`

**Out of scope**:

- Domain and application service logic across all modules.

## Git workflow

- Branch: `advisor/003-http-limits-observability-and-headers`
- Commit message style: `fix(httpx): bound request bodies, log 500 errors, and set security headers`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Bound `DecodeJSON`, add `X-Content-Type-Options: nosniff`, and add logging/middleware in `httpx` (`SEC-04`, `SEC-07`, `DX-03`)

1. In `api/internal/platform/httpx/response.go`:
   - Define `const DefaultMaxJSONBytes = 2 << 20` (2 MiB) and `const MaxUploadBodyBytes = 25 << 20` (25 MiB).
   - In `JSON(w, status, payload)` and `NoContent(w)`, set `w.Header().Set("X-Content-Type-Options", "nosniff")`.
   - In `DecodeJSON(r, dst)`, wrap `r.Body` with `io.LimitReader(r.Body, DefaultMaxJSONBytes+1)` (or add `DecodeJSONWithLimit(r, dst, maxBytes)` used by `DecodeJSON` with `DefaultMaxJSONBytes`) and return an error if more than `maxBytes` bytes are read.
   - Add `LogInternalError(w http.ResponseWriter, message string, err error)` that calls `slog.Error(message, slog.Any("error", err))` and then `InternalError(w, message)`.
   - Add a `RequestLogger(next http.Handler) http.Handler` middleware in `httpx` that recovers from panics (logging the panic and returning 500) and logs `method`, `path`, `status`, and `duration_ms` via `slog.Info`.

**Verify**: `cd api && go test ./internal/platform/httpx/...` → exit 0.

### Step 2: Update HTTP handlers for upload body limits, `Cache-Control: no-store`, FEC filename sanitization, and 500 logging (`SEC-04`, `SEC-07`, `DX-03`)

1. In all 7 `interfaces/http/handler.go` files (`auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, `accounting`), replace `httpx.InternalError(w, "internal <module> error")` in the `default:` case of `writeDomainError` with `httpx.LogInternalError(w, "internal <module> error", err)`.
2. In `api/internal/modules/auth/interfaces/http/handler.go`:
   - In `handleLogin`, `handleRefresh`, and `handleCreateMCPToken`, set `w.Header().Set("Cache-Control", "no-store")` before writing the token response.
3. In `api/internal/modules/payment/interfaces/http/handler.go:85-151` (`handleImportPaymentsCSV`):
   - Apply `r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxUploadBodyBytes)` at the top of `handleImportPaymentsCSV`, and limit `io.ReadAll(file)` / `io.ReadAll(r.Body)` with `io.LimitReader(..., httpx.MaxUploadBodyBytes)`.
4. In `api/internal/modules/document/interfaces/http/handler.go`:
   - In `handleRegisterTemplate` (line 46) and the JSON branch of `handleAttachFile` (line 141), use `r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxUploadBodyBytes)` and `httpx.DecodeJSONWithLimit(r, &req, httpx.MaxUploadBodyBytes)`.
5. In `api/internal/modules/accounting/interfaces/http/handler.go:146-148` (`handleExportFEC`):
   - Sanitize `entityID` so only `[a-zA-Z0-9_-]` characters are included in `Content-Disposition` `filename="..."` and set `w.Header().Set("X-Content-Type-Options", "nosniff")`.

**Verify**: `cd api && go vet ./... && go test ./...` → exit 0.

### Step 3: Configure `slog` JSON logger in `main.go` and `http.Server` timeouts in `server.go` (`SEC-04`, `DX-03`)

1. In `api/cmd/main.go`, configure `slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))` at the start of `main()`.
2. In `api/cmd/server/server.go`:
   - Wrap the returned root `mux` in `buildHTTPHandler` with `httpx.RequestLogger(mux)`.
   - In `serveHTTP` (lines 129–133), set `ReadTimeout: 30 * time.Second`, `WriteTimeout: 60 * time.Second`, and `IdleTimeout: 120 * time.Second` on `http.Server`.

**Verify**: `cd api && go vet ./... && go test ./...` → exit 0.

## Test plan

- Create `api/internal/platform/httpx/response_test.go` testing:
  - `DecodeJSON` rejects payloads larger than `DefaultMaxJSONBytes` (2 MiB).
  - `JSON` sets `X-Content-Type-Options: nosniff`.
  - `RequestLogger` logs requests and recovers from handler panics with HTTP 500.
- Run `cd api && go test -v ./internal/platform/httpx/... ./cmd/...` → all pass.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test ./...` exits 0
- [ ] `httpx.DecodeJSON` rejects oversized payloads in `response_test.go`
- [ ] `grep -rn 'httpx.InternalError(w, "internal ' api/internal/modules/` returns 0 matches (all replaced with `httpx.LogInternalError`)
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If any file in "Current state" has drifted from the excerpts above.
- If a step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- Standard JSON command endpoints inherit the 2 MiB limit automatically via `httpx.DecodeJSON`; any future endpoint accepting large binary/base64 payloads should explicitly call `httpx.DecodeJSONWithLimit(r, &dst, httpx.MaxUploadBodyBytes)`.
