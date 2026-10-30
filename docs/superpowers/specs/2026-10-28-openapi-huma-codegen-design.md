# Design Spec — End-to-End Go-First OpenAPI 3.1 (Huma v2) & Auto-Generated SvelteKit SSR Client

**Date:** 2026-10-28
**Status:** Draft (Pending User Approval)
**Scope:** `api/internal/platform/httpx`, `api/internal/modules/*/interfaces/http`, `api/cmd/server`, `web/src/lib/core/api`, `web/src/lib/server/backend-api.ts`

---

## 1. Intent & Why

Whenever a backend route path, query parameter, JSON field name, or domain enum changes in the Go API (`api/`), maintaining hand-written TypeScript DTOs (`web/src/lib/modules/*/types.ts`) and hand-written endpoint wrappers (`web/src/lib/server/backend-api.ts`) causes silent drift (e.g., `ENTRANT` vs. `MOVE_IN`, `FAIR` vs. `FAIR_WEAR`, `RESIDENTIAL_EMPTY` vs. `RESIDENTIAL_UNFURNISHED`).

At the same time, writing raw OpenAPI YAML by hand or introducing a separate IDL compiler (Protobuf/GraphQL/TypeSpec) adds tooling friction and duplicates definitions that already exist in Go structs.

**Solution:** Make the Go HTTP layer the **single source of truth** using **Huma v2 (`github.com/danielgtaylor/huma/v2` + `humago` adapter on standard `net/http.ServeMux`)**, export the OpenAPI 3.1 specification deterministically without requiring a running database, and generate the TypeScript schema and typed client in `web/` via `openapi-typescript`.

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
1. **Zero Manual OpenAPI YAML:** All OpenAPI 3.1 paths, parameters, request bodies, response bodies, status codes, and enum constraints are derived directly from Go input/output structs registered with Huma v2.
2. **Zero Drift Between Go Handlers and OpenAPI Spec:** The Go handler signatures (`func(ctx context.Context, input *In) (*Out, error)`) *are* the OpenAPI contract. A handler cannot read a path/query/body field without declaring it in its `In` struct.
3. **Deterministic Offline Schema Export:** Running `make -C api openapi` (which executes `go run ./cmd server --export-openapi ../web/src/lib/core/api/openapi.json`) outputs the complete OpenAPI 3.1 JSON spec using in-memory module wiring (`pool == nil`), requiring no PostgreSQL or Gotenberg instance.
4. **End-to-End Type Safety in SvelteKit SSR:** Running `npm --prefix web run gen:api` (wired into `Makefile` and verifiable in CI) generates `web/src/lib/core/api/openapi.d.ts` from `openapi.json`. `web/src/lib/server/backend-api.ts` and `web/src/lib/modules/*/types.ts` derive their types directly from `components['schemas']` and `paths`, so any backend contract change immediately surfaces in `npm --prefix web run check` (`svelte-check`).
5. **100% Backward Compatibility with Existing Tests & Error Semantics:** Keep the existing `{"error": "<message>"}` JSON error envelope (alongside RFC 9457 / Huma error handling) so `ApiError` in `web/` and all Go HTTP tests continue to work seamlessly.

### Non-Goals
- Replacing the Streamable HTTP MCP server (`POST /mcp`, `GET /mcp`, `GET /.well-known/oauth-protected-resource`). MCP routes remain mounted directly on the root `http.ServeMux`.
- Modifying domain entities, application services, or PostgreSQL/memory repositories.

### Anti-Goals (What Would Make This a Failure)
- Introducing a heavy external codegen daemon, Java/Docker generator, or custom IDL language.
- Polluting `domain` or `application` packages with HTTP/Huma imports (strictly forbidden by `AGENTS.md` Dependency Inversion Principle).
- Breaking `AGENTS.md` Rule 4 (`api/cmd/main.go` must only assemble `server.Command()`, `bootstrap.Command()`, and `token.Command()`).

---

## 3. Architecture & Technical Decisions

### 3.1 Go Backend (`api/`) — Huma v2 on Standard `http.ServeMux`

1. **Platform Helper (`api/internal/platform/httpx/huma.go`)**:
   - Configures `huma.NewConfig("Gestion Immo API", "1.0.0")` with `humago.New(mux, cfg)`.
   - Overrides Huma's default error formatter (`huma.NewError`) to return a unified error struct that serializes `{"error": "...", "status": 422, "title": "...", "detail": "..."}` — preserving the exact `{"error": "..."}` field expected by `web/src/lib/core/api/http-client.ts` (`extractErrorMessage`) and existing Go HTTP tests, while remaining valid OpenAPI error responses.
   - Provides a central domain-error-to-Huma-StatusError helper (`httpx.MapDomainError`) or per-module error mapper (`400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `409 Conflict`, `422 Unprocessable Entity`, `500 Internal Server Error`).

2. **Module Route Registration (`api/internal/modules/<module>/interfaces/http/handler.go`)**:
   - Each module's HTTP `Handler` exposes `RegisterHumaRoutes(api huma.API)` (and keeps `RegisterRoutes(mux *http.ServeMux)` as a thin wrapper that creates or delegates to a `huma.API` instance so standalone module HTTP tests continue to work without changes).
   - Every operation has an explicit `OperationID` (e.g., `listEntities`, `createEntity`, `getEntitySummary`, `upsertShareholder`, `listProperties`, `createProperty`, `createLot`, `listTenants`, `createTenant`, `archiveTenant`, `listLeases`, `createLease`, `generateRentCalls`, `listEntityRentCalls`, `recordCashPayment`, `recordBankPayment`, `recordCashDeposit`, `importPaymentsCsv`, `listNegativePositions`, `listCashJournal`, `listDocumentTemplates`, `registerDocumentTemplate`, `generateDocumentFromTemplate`, `uploadDocumentFile`, `assembleGliDossier`, `listLeaseDocuments`, `listInspectionTemplates`, `createInspectionTemplate`, `getInspectionTemplate`, `listInspections`, `createInspection`, `getInspection`, `upsertInspectionItems`, `generateInspectionPdf`, `listIncidents`, `createIncident`, `getIncident`, `updateIncident`, `listJournalEntries`, `recordJournalEntry`, `upsertBalanceLines`, `previewStatements`, `generateAnnualAccountsPdf`, `exportFecTsv`, `signup`, `login`, `refresh`, `logout`, `createMcpToken`, `listPendingUsers`, `approveUser`, `rejectUser`).
   - **Enum & Validation Tags:** Input structs and DTOs declare `enum:"..."` tags where applicable (e.g., `InspectionType`: `enum:"MOVE_IN,MOVE_OUT"`, `ConditionState`: `enum:"NEW,GOOD,FAIR_WEAR,POOR,OUT_OF_SERVICE"`, `IncidentStatus`: `enum:"OPEN,IN_PROGRESS,RESOLVED,CLOSED"`, `LeaseType`: `enum:"RESIDENTIAL_UNFURNISHED,RESIDENTIAL_FURNISHED,MOBILITY,COMMERCIAL_3_6_9,PROFESSIONAL"`, `LegalEntityType`: `enum:"SCI_IR,SCI_IS,SARL_FAMILLE,SARL_IS,LMNP,LMP,OWN_NAME"`, `TemplateType`: `enum:"LEASE_CONTRACT,RENT_RECEIPT,FORMAL_NOTICE,GLI_DECLARATION,CUSTOM"`, `TemplateFormat`: `enum:"DOCX,HTML"`).
   - **Special Content-Types:**
     - `GET /api/v1/entities/{id}/fec`: Uses Huma's `huma.StreamResponse` (with `Content-Type: text/tab-separated-values; charset=utf-8` and `Content-Disposition` header) while documenting a `string` response schema in OpenAPI.
     - `POST /api/v1/documents/upload` & `POST /api/v1/payments/import-csv`: Support both `multipart/form-data` (`huma.MultipartFormFiles`) and JSON payloads seamlessly.

3. **Authentication & Security Scheme in OpenAPI**:
   - Registers the `bearerAuth` HTTP Bearer JWT security scheme in `api.OpenAPI().Components.SecuritySchemes`.
   - Public auth routes (`POST /api/v1/auth/signup`, `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`) have no security requirement.
   - All `/api/v1/...` protected routes enforce the existing JWT `RequireAuthMiddleware` (and `RequireSuperAdmin` on `/api/v1/admin/users/...`) via a Huma middleware / `http.ServeMux` wrapper.

4. **CLI Export Flag in `api/cmd/server/server.go`**:
   - Adds `--export-openapi <file-path>` flag to `server.Command()` (keeping `api/cmd/main.go` strictly limited to `server`, `bootstrap`, and `token` per `AGENTS.md` Rule 4).
   - When `--export-openapi` is provided, `server` builds the Huma OpenAPI spec in-memory (no DB connection opened), marshals `api.OpenAPI()` to formatted JSON (or writes to stdout if `-`), and exits cleanly with code `0`.
   - Adds `openapi:` target to `api/Makefile`:
     ```makefile
     openapi: ## Export OpenAPI 3.1 specification to ../web/src/lib/core/api/openapi.json
     	go run ./cmd server --export-openapi ../web/src/lib/core/api/openapi.json
     ```

---

### 3.2 SvelteKit SSR Frontend (`web/`) — `openapi-typescript` + `openapi-fetch` (Zero Custom `HttpClient`)

1. **Type Generation & Official Client (`openapi-typescript` + `openapi-fetch`)**:
   - Installs `openapi-typescript` (`devDependency`) and `openapi-fetch` (`dependency`, 2 KB zero-dependency wrapper around standard `fetch`) in `web/package.json`.
   - Deletes the custom `web/src/lib/core/api/http-client.ts` and `web/src/lib/core/api/http-client.test.ts` — `openapi-fetch` (`createClient<paths>`) replaces `FetchHttpClient` completely.
   - Adds script `"gen:api": "openapi-typescript ./src/lib/core/api/openapi.json -o ./src/lib/core/api/openapi.d.ts && prettier --write ./src/lib/core/api/openapi.d.ts"` to `web/package.json`.
   - Commits both `web/src/lib/core/api/openapi.json` and `web/src/lib/core/api/openapi.d.ts` so `npm --prefix web run check` and Cloudflare Worker builds run deterministically in any environment.

2. **Single Source of Types (`web/src/lib/modules/*/types.ts`)**:
   - Each module's `types.ts` re-exports its DTOs, command payloads, and enums directly from `components['schemas']` in `#lib/core/api/openapi.d.ts` (e.g., `export type InspectionDTO = components['schemas']['InspectionDTO'];`).
   - Also fixes existing enum drift discovered in `leases/+page.svelte` (`RESIDENTIAL_EMPTY` → `RESIDENTIAL_UNFURNISHED`, `COMMERCIAL` → `COMMERCIAL_3_6_9`) and `entities/+page.svelte` (`SCI_IR`, `SCI_IS`, `SARL_FAMILLE`, `SARL_IS`, `LMNP`, `LMP`, `OWN_NAME`) so that UI options match the Go domain enums 100%.

3. **`openapi-fetch` Server Client (`web/src/lib/server/backend-api.ts`)**:
   - `createBackendApi(event)` instantiates `createClient<paths>({ baseUrl, fetch: event.fetch, headers: defaultHeaders })` from `openapi-fetch` with an `openapi-fetch` middleware (`client.use(...)`) that injects the `Authorization: Bearer <accessToken>` header and transparently refreshes the session on `401` (updating `HttpOnly` cookies via `setSessionCookies` or clearing via `clearSessionCookies`).
   - Every route path, path parameter (`params.path`), query parameter (`params.query`), request body (`body`), and response payload in `backend-api.ts` (or direct `locals.api.client` calls) is 100% statically typed against `paths` from `#lib/core/api/openapi.d.ts`.

---

## 4. Verification Criteria

1. `make -C api openapi` generates `web/src/lib/core/api/openapi.json` with all 42 REST operations and component schemas.
2. `npm --prefix web run gen:api` generates `web/src/lib/core/api/openapi.d.ts` cleanly.
3. `make -C api test && make -C api lint && make -C api build` passes with 0 errors.
4. `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check && npm --prefix web run build` passes with 0 errors and 0 warnings.
