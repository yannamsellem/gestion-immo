# Go-First OpenAPI 3.1 (Huma v2) & `openapi-fetch` SvelteKit SSR Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all Go REST handlers to Huma v2 (`humago` on `net/http.ServeMux`), export deterministic OpenAPI 3.1 JSON via `server --export-openapi`, generate `web/src/lib/core/api/openapi.d.ts` via `openapi-typescript`, and replace the custom `FetchHttpClient` (`http-client.ts`) and manual DTO definitions with `openapi-fetch`.

**Architecture:** Huma v2 wraps the Go 1.22+ `http.ServeMux` in `api/internal/platform/httpx/huma.go` with a custom error formatter that preserves the `{"error": "..."}` JSON contract alongside RFC 9457 status codes. Each bounded context's `interfaces/http/handler.go` registers typed Huma operations (`RegisterHumaRoutes(api huma.API)`), while `server --export-openapi` dumps the OpenAPI 3.1 JSON spec offline (using `pool == nil` in-memory wiring). On the SvelteKit SSR side, `openapi-typescript` generates `openapi.d.ts` and `openapi-fetch` replaces `FetchHttpClient` completely.

**Tech Stack:** Go 1.26, `github.com/danielgtaylor/huma/v2` (`humago`), SvelteKit 5 SSR (`@sveltejs/kit`), `openapi-typescript`, `openapi-fetch`, Vitest.

**Spec:** `docs/superpowers/specs/2026-10-28-openapi-huma-codegen-design.md`

## Global Constraints

- **100% English in Code (`AGENTS.md` Rule 1):** All Go identifiers, struct fields, JSON tags, OpenAPI `operationId`s, and TypeScript types/variables must be 100% in English.
- **Monetary Arithmetic (`AGENTS.md` Rule 2):** All monetary fields remain `money.AmountCents` (`int64`) and percentages remain `money.BasisPoints` (`int64`).
- **DIP & Clean Architecture (`AGENTS.md` Rule 5):** `huma` imports are strictly confined to `api/internal/platform/httpx`, `api/internal/modules/*/interfaces/http`, and `api/cmd/server`. Never import `huma` in `domain` or `application`.
- **CLI Root Commands (`AGENTS.md` Rule 4):** `api/cmd/main.go` must only assemble `server.Command()`, `bootstrap.Command()`, and `token.Command()`. OpenAPI export is exposed via `--export-openapi <path>` on `server.Command()`.
- **Mandatory Web Formatting (`AGENTS.md` Rule 5):** Whenever any `.ts`, `.svelte`, or `.css` file in `web/` is created or modified, ALWAYS run `npm --prefix web run format` immediately afterward.

## Review Focus

1. **Error payload backward compatibility (`{"error": "..."}`):** Huma's default RFC 9457 error marshaler omits the top-level `"error"` key; `httpx.NewHumaAPI` must override `huma.NewError` so every 4xx/5xx error JSON includes `"error": "<detail>"` alongside `"status"` and `"detail"`.
2. **Nil slice serialization (`[]` vs `null`):** Go `nil` slices inside Huma output `Body` structs serialize to `null` unless initialized to `[]T{}`; every list handler must ensure non-nil empty slices `[]T{}` before returning `&ListOutput{Body: list}`.
3. **Dual-mode payload endpoints (`POST /api/v1/documents/upload`, `POST /api/v1/payments/import-csv`):** Existing callers and tests send either `multipart/form-data` or `application/json`; handlers must continue supporting both while exposing clean typed schemas in OpenAPI.
4. **Optional JSON bodies on POST endpoints (`POST /api/v1/entities/{id}/rent-calls/generate`, `POST /api/v1/entities/{id}/statements/preview`, `POST /api/v1/entities/{id}/statements/pdf`, `POST /api/v1/leases/{id}/gli-dossier`, `POST /api/v1/auth/mcp-tokens`):** Callers may send an empty body (`Content-Length: 0`) or a JSON body; Huma input structs or raw body readers must accept empty bodies without failing with `400 EOF`.
5. **Enum alignment in Svelte UI (`leases/+page.svelte`, `entities/+page.svelte`):** Replacing manual `types.ts` with generated OpenAPI union types will flag any outdated enum literals in `.svelte` files (`RESIDENTIAL_EMPTY` → `RESIDENTIAL_UNFURNISHED`, `COMMERCIAL` → `COMMERCIAL_3_6_9`, `SCIA` → `SCI_IR` / `SCI_IS`); update those UI select options so `svelte-check` passes with 0 errors.

---

### Task 1: Platform Huma v2 Adapter & Unified Error Envelope (`api/internal/platform/httpx`)

**Files:**
- Modify: `api/go.mod`, `api/go.sum`
- Create: `api/internal/platform/httpx/huma.go`
- Test: `api/internal/platform/httpx/httpx_test.go`

**Interfaces:**
- Consumes: `github.com/danielgtaylor/huma/v2`, `github.com/danielgtaylor/huma/v2/adapters/humago`
- Produces:
  - `func NewHumaAPI(mux *http.ServeMux) huma.API`
  - `type ErrorModel struct { Status int `json:"status"`; Title string `json:"title,omitempty"`; Detail string `json:"detail,omitempty"`; Error string `json:"error"` }` (implements `huma.StatusError`)
  - `func StatusError(status int, message string) huma.StatusError`

- [ ] **Step 1: Add `github.com/danielgtaylor/huma/v2` to `api/go.mod` and write the failing test in `api/internal/platform/httpx/httpx_test.go`**

```go
func TestNewHumaAPI_PreservesErrorJSONEnvelopeAndExportsOpenAPI(t *testing.T) {
	mux := http.NewServeMux()
	api := httpx.NewHumaAPI(mux)

	type pingOutput struct {
		Body struct {
			Status string `json:"status"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "pingTest",
		Method:      http.MethodGet,
		Path:        "/api/v1/ping",
	}, func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		return nil, httpx.StatusError(http.StatusUnprocessableEntity, "invalid domain input")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error payload: %v", err)
	}
	if payload["error"] != "invalid domain input" {
		t.Fatalf("expected error field 'invalid domain input', got %v", payload["error"])
	}
	if api.OpenAPI().Paths["/api/v1/ping"] == nil {
		t.Fatal("expected /api/v1/ping in OpenAPI spec")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -C api ./internal/platform/httpx/...`
Expected: FAIL (`undefined: httpx.NewHumaAPI`)

- [ ] **Step 3: Implement `NewHumaAPI`, `ErrorModel`, and `StatusError` in `api/internal/platform/httpx/huma.go`**

Configure `huma.NewError` inside `init()` / `NewHumaAPI` so validation errors and `httpx.StatusError` always populate `Error: detail` in JSON, register the `bearerAuth` HTTP JWT security scheme on `api.OpenAPI().Components.SecuritySchemes`, and disable default schema prefix collisions if any.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -C api ./internal/platform/httpx/...`
Expected: PASS

---

### Task 2: Migrate `auth`, `portfolio`, `lease`, and `payment` HTTP Handlers to Huma v2

**Files:**
- Modify: `api/internal/modules/auth/interfaces/http/handler.go`
- Modify: `api/internal/modules/auth/module.go`
- Modify: `api/internal/modules/portfolio/interfaces/http/handler.go`
- Modify: `api/internal/modules/portfolio/module.go`
- Modify: `api/internal/modules/lease/interfaces/http/handler.go`
- Modify: `api/internal/modules/lease/module.go`
- Modify: `api/internal/modules/payment/interfaces/http/handler.go`
- Modify: `api/internal/modules/payment/module.go`

**Interfaces:**
- Consumes: `httpx.NewHumaAPI`, `httpx.StatusError`, `huma.API`, `huma.Register`
- Produces:
  - `func (h *Handler) RegisterHumaRoutes(api huma.API)` and `func (m *Module) RegisterHumaRoutes(api huma.API)` on `auth`, `portfolio`, `lease`, and `payment`.
  - `func (h *Handler) RegisterRoutes(mux *http.ServeMux)` delegates to `h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))` so all existing unit/integration tests using `RegisterRoutes(mux)` continue to work untouched.

- [ ] **Step 1: Implement `RegisterHumaRoutes(api huma.API)` in `auth`, `portfolio`, `lease`, and `payment` HTTP handlers**

Ensure:
- `auth`: operations `signup` (`202 Accepted`), `login` (`200 OK`, `Cache-Control: no-store`), `refresh` (`200 OK`), `logout` (`204 No Content`), `createMcpToken` (`201 Created`, optional JSON body), `listPendingUsers` (`200 OK`), `approveUser` (`200 OK`), `rejectUser` (`200 OK`).
- `portfolio`: operations `createEntity` (`201`), `listEntities` (`200`), `getEntitySummary` (`200`), `upsertShareholder` (`200`), `createProperty` (`201`), `listProperties` (`200`), `createLot` (`201`).
- `lease`: operations `createTenant` (`201`), `listTenants` (`200`, query `include_archived`), `archiveTenant` (`204`), `createLease` (`201`), `listLeases` (`200`, query `entity_id` / `legal_entity_id`), `generateRentCalls` (`201`, optional JSON body or query `period_month`), `listEntityRentCalls` (`200`, optional query `period_month`).
- `payment`: operations `recordCashPayment` (`201`), `recordBankPayment` (`201`), `recordCashDeposit` (`201`), `importPaymentsCsv` (`200`, supports JSON `{entity_id, csv_content}` and `multipart/form-data`), `listNegativePositions` (`200`), `listCashJournal` (`200`).

- [ ] **Step 2: Run Go tests for `auth`, `portfolio`, `lease`, and `payment`**

Run: `go test -C api ./internal/modules/auth/... ./internal/modules/portfolio/... ./internal/modules/lease/... ./internal/modules/payment/...`
Expected: PASS

---

### Task 3: Migrate `document`, `inspection`, and `accounting` HTTP Handlers + `server --export-openapi`

**Files:**
- Modify: `api/internal/modules/document/interfaces/http/handler.go`
- Modify: `api/internal/modules/document/module.go`
- Modify: `api/internal/modules/inspection/interfaces/http/handler.go`
- Modify: `api/internal/modules/inspection/module.go`
- Modify: `api/internal/modules/accounting/interfaces/http/handler.go`
- Modify: `api/internal/modules/accounting/module.go`
- Modify: `api/cmd/server/server.go`
- Modify: `api/cmd/main_test.go`
- Modify: `api/Makefile`

**Interfaces:**
- Consumes: `httpx.NewHumaAPI`, `httpx.StatusError`, `huma.API`, `huma.StreamResponse`
- Produces:
  - `func (h *Handler) RegisterHumaRoutes(api huma.API)` and `func (m *Module) RegisterHumaRoutes(api huma.API)` on `document`, `inspection`, and `accounting`.
  - `func BuildOpenAPI() (*huma.OpenAPI, error)` in `api/cmd/server/server.go`.
  - CLI flag `--export-openapi <path>` on `server.Command()` and `make -C api openapi`.

- [ ] **Step 1: Add OpenAPI export test in `api/cmd/main_test.go`**

```go
func TestServer_ExportOpenAPIFlag_WritesValidOpenAPI31SpecWithoutDatabase(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "openapi.json")
	app := &cli.Command{
		Name:     "gestion-immo",
		Commands: []*cli.Command{server.Command()},
	}
	if err := app.Run(context.Background(), []string{"gestion-immo", "server", "--export-openapi", outPath}); err != nil {
		t.Fatalf("export-openapi failed: %v", err)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exported openapi.json: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("invalid openapi json: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %v", doc["openapi"])
	}
	paths, _ := doc["paths"].(map[string]any)
	for _, requiredPath := range []string{
		"/api/v1/auth/login",
		"/api/v1/entities",
		"/api/v1/leases",
		"/api/v1/payments/cash",
		"/api/v1/documents/upload",
		"/api/v1/inspections",
		"/api/v1/entities/{id}/fec",
	} {
		if paths[requiredPath] == nil {
			t.Fatalf("missing required path %s in exported OpenAPI spec", requiredPath)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -C api ./cmd/... -run TestServer_ExportOpenAPIFlag`
Expected: FAIL (`flag provided but not defined: -export-openapi`)

- [ ] **Step 3: Implement `RegisterHumaRoutes` for `document`, `inspection`, `accounting` and `--export-openapi` in `api/cmd/server/server.go`**

Ensure:
- `document`: operations `registerDocumentTemplate` (`201`), `listDocumentTemplates` (`200`), `generateDocumentFromTemplate` (`201`), `uploadDocumentFile` (`201`, supports `multipart/form-data` and JSON base64), `assembleGliDossier` (`201`, optional body), `assembleGliDossierBody` (`201`), `listLeaseDocuments` (`200`).
- `inspection`: operations `createInspectionTemplate` (`201`), `listInspectionTemplates` (`200`), `getInspectionTemplate` (`200`), `createInspection` (`201`), `listInspections` (`200`, query `lease_id`), `getInspection` (`200`), `upsertInspectionItems` (`200`, accepts `{items: [...]}` or `[...]`), `generateInspectionPdf` (`201`), `createIncident` (`201`), `listIncidents` (`200`, query `property_id`, `lease_id`), `getIncident` (`200`), `updateIncident` (`200`).
- `accounting`: operations `recordJournalEntry` (`201`), `listJournalEntries` (`200`, query `fiscal_year`), `upsertBalanceLines` (`204`), `previewStatements` (`200`, optional body), `generateAnnualAccountsPdf` (`201`, optional body), `exportFecTsv` (`200`, `text/tab-separated-values; charset=utf-8` via `huma.StreamResponse`).
- `server.go`: wire all modules onto a single `huma.API` in `BuildHTTPHandler` (preserving `RequireAuthMiddleware` on protected `/api/v1/` routes and `RequireSuperAdmin` on `/api/v1/admin/users/...`), add `--export-openapi` flag to `server.Command()`, and add `openapi` target to `api/Makefile`.

- [ ] **Step 4: Run all Go tests, linter, build, and export `web/src/lib/core/api/openapi.json`**

Run: `make -C api test && make -C api lint && make -C api build && make -C api openapi`
Expected: PASS and `web/src/lib/core/api/openapi.json` created.

---

### Task 4: Generate `openapi.d.ts`, Replace `FetchHttpClient` with `openapi-fetch`, and Align Module Types (`web/`)

**Files:**
- Create: `web/src/lib/core/api/openapi.json`
- Create: `web/src/lib/core/api/openapi.d.ts`
- Delete: `web/src/lib/core/api/http-client.ts`
- Delete: `web/src/lib/core/api/http-client.test.ts`
- Create: `web/src/lib/server/backend-api.test.ts`
- Modify: `web/package.json`
- Modify: `web/src/lib/server/backend-api.ts`
- Modify: `web/src/lib/modules/auth/types.ts`
- Modify: `web/src/lib/modules/portfolio/types.ts`
- Modify: `web/src/lib/modules/lease/types.ts`
- Modify: `web/src/lib/modules/payment/types.ts`
- Modify: `web/src/lib/modules/document/types.ts`
- Modify: `web/src/lib/modules/inspection/types.ts`
- Modify: `web/src/lib/modules/accounting/types.ts`
- Modify: `web/src/routes/(app)/leases/+page.svelte`
- Modify: `web/src/routes/(app)/leases/+page.server.ts`
- Modify: `web/src/routes/(app)/entities/+page.svelte`

**Interfaces:**
- Consumes: `web/src/lib/core/api/openapi.json`, `openapi-typescript`, `openapi-fetch`
- Produces:
  - `web/src/lib/core/api/openapi.d.ts` (`paths`, `components`)
  - `createBackendApi(event)` in `web/src/lib/server/backend-api.ts` powered by `createClient<paths>` from `openapi-fetch` (with `client.use(...)` middleware for JWT Bearer header, Cloudflare Access headers, and automatic `401` token refresh).

- [ ] **Step 1: Install `openapi-typescript` and `openapi-fetch` in `web/` and run `npm --prefix web run gen:api`**

Run: `npm --prefix web install openapi-fetch && npm --prefix web install -D openapi-typescript && npm --prefix web run gen:api`
Expected: `web/src/lib/core/api/openapi.d.ts` generated and formatted.

- [ ] **Step 2: Delete `web/src/lib/core/api/http-client.ts` and `http-client.test.ts`, and write `web/src/lib/server/backend-api.test.ts`**

Test that `createBackendApi`:
1. Sends `Authorization: Bearer <token>` and `CF-Access-Client-Id` / `CF-Access-Client-Secret` via `openapi-fetch` middleware.
2. Automatically calls `/api/v1/auth/refresh` on `401`, updates cookies, and retries the original request once.
3. Throws `ApiError` with the backend's `{"error": "..."}` message on non-2xx responses.

- [ ] **Step 3: Refactor `web/src/lib/modules/*/types.ts` to derive types from `components['schemas']` in `openapi.d.ts` and refactor `web/src/lib/server/backend-api.ts` to use `createClient<paths>` (`openapi-fetch`)**

Also align any enum literals in `web/src/routes/(app)/leases/+page.svelte`, `web/src/routes/(app)/leases/+page.server.ts`, and `web/src/routes/(app)/entities/+page.svelte` (`RESIDENTIAL_UNFURNISHED`, `RESIDENTIAL_FURNISHED`, `MOBILITY`, `COMMERCIAL_3_6_9`, `PROFESSIONAL`; `SCI_IR`, `SCI_IS`, `SARL_FAMILLE`, `SARL_IS`, `LMNP`, `LMP`, `OWN_NAME`).

- [ ] **Step 4: Run mandatory formatting, linting, type-checking, unit tests, and production build**

Run: `npm --prefix web run format && npm --prefix web run check && npm --prefix web run lint && npm --prefix web test && npm --prefix web run build`
Expected: PASS with 0 errors and 0 warnings.

- [ ] **Step 5: Commit all changes**

```bash
git add api/ web/ docs/
git commit -m "feat(api,web): adopt Go-first OpenAPI 3.1 with Huma v2 and openapi-fetch SSR client"
```
