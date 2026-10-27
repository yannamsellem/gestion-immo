# `gestion-immo` Web Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the complete Svelte 5 (`runes`) + `shadcn-svelte` (`mira` / `zinc` / `tabler`) multi-entity real-estate management web frontend in `web/`, centered on an operational Cockpit by Legal Entity and Property with inline CTAs and contextual right-hand `Sheet` workflows.

**Architecture:** Clean, modular frontend architecture under `web/src/lib/` (`#lib/*` subpath alias) following SOLID principles: a transport-agnostic `HttpClient` with single-flight JWT refresh (`#lib/core/api`), pure integer cent/basis-point formatters (`#lib/core/formatters`), per-domain typed API services (`#lib/modules/*`), a pure Cockpit aggregation engine (`#lib/modules/cockpit`), and Svelte 5 rune-driven routes/components composed from official `shadcn-svelte` CLI primitives and `mode-watcher`.

**Tech Stack:** Svelte `^5.57.1` (strict runes mode), SvelteKit `^3.0.0` (`@sveltejs/adapter-cloudflare`, SPA mode `ssr = false`), Tailwind CSS `^4.3.0`, `shadcn-svelte` `^1.7.1` (`mira` style, `zinc` base color, `@tabler/icons-svelte`), `mode-watcher`, `vitest`.

**Spec:** `docs/superpowers/specs/2026-10-27-web-frontend-design.md`

## Global Constraints

- **Strictly 100% English in Code (`AGENTS.md`):** All files, directories, route slugs, TypeScript interfaces, functions, variables, props, and tests MUST be 100% in English. French is strictly confined to UI display labels and formatting output.
- **No Floating-Point Money (`AGENTS.md`):** API DTOs exclusively transport raw integer cents (`*_cents: number`) and basis points (`*_basis_points: number` where `10000 = 100.00%`). Parse user inputs into integer cents at the form boundary (`parseEuroToCents`) and format only for display (`formatCentsToEuro`).
- **Subpath Import Aliases (`web/package.json` & `web/components.json`):** Always use `#lib`, `#lib/components`, `#lib/components/ui`, `#lib/utils`, `#lib/hooks` (never `$lib`).
- **Svelte 5 Runes Only (`web/vite.config.ts`):** Enforced by compiler options. Use `$state`, `$derived`, `$effect`, `$props`, `$bindable`, `{#snippet}`, `{@render}`, and standard `onclick` attributes. Never use `export let`, `$:`, `<slot>`, or `on:click`.
- **`shadcn-svelte` Composition & Dark Mode Rules:** Install UI primitives via `npx shadcn-svelte@latest add`, use `mode-watcher` (`<ModeWatcher />` + `toggleMode` / `setMode` / `resetMode`), use `@tabler/icons-svelte` with `data-icon="inline-start" | "inline-end"` inside buttons (no manual `size-*` on icons inside buttons/menus), use `Field.FieldGroup` + `Field.Field` for forms, `Select.Group` inside `Select.Content`, `Empty.*` for empty states, and semantic color tokens without manual `dark:` overrides.

## Review Focus

1. **French formatted euro & percentage inputs with spaces, commas, or invalid values:** `parseEuroToCents("1 234,56 €")` must return `123456` and reject negative/malformed strings with an explicit error instead of producing `NaN` or floating-point rounding drift (tested in Task 1 `money.test.ts`).
2. **Concurrent `401 Unauthorized` responses during parallel Cockpit fetches:** `FetchHttpClient` must coalesce concurrent token refresh attempts into a single in-flight `POST /api/v1/auth/refresh` request so token rotation does not invalidate parallel requests, and clear the session if refresh fails (tested in Task 1 `http-client.test.ts`).
3. **Properties/lots with no active lease or multiple simultaneous pending tasks on the Cockpit:** `buildCockpitEntityGroups` must include vacant lots with a `"Créer un bail"` CTA and order multiple pending CTAs on leased lots deterministically (`remind` > `pay` > `generate-calls` > `incident`) (tested in Task 2 `cockpit-builder.test.ts`).
4. **Rapid toggling of financial statement lines on `/accounting`:** Toggling a statement line must deterministically add/remove its `line_id` in `disabledLineIds` without duplicates and accurately compute active vs. disabled counts (tested in Task 6 `statement-state.test.ts`).
5. **Cockpit CTA deep-links (`?entityId=...&leaseId=...&action=pay`) when Workspace is set to `"ALL"`:** Target pages (`/rents`, `/inspections`) must sync `workspace.selectedEntityId` from the `entityId` query parameter and auto-open the right-hand `Sheet` for the targeted record (implemented in Tasks 4 & 5).

---

### Task 1: Project Tooling, `shadcn-svelte` UI Primitives, `mode-watcher`, Formatters & HTTP Client

**Files:**
- Modify: `web/package.json`
- Modify: `web/vite.config.ts`
- Create: `web/src/routes/+layout.ts`
- Create: `web/src/lib/core/formatters/money.ts`
- Create: `web/src/lib/core/formatters/date.ts`
- Create: `web/src/lib/core/api/api-error.ts`
- Create: `web/src/lib/core/api/http-client.ts`
- Test: `web/src/lib/core/formatters/money.test.ts`
- Test: `web/src/lib/core/api/http-client.test.ts`

**Interfaces:**
- Produces:
  - `formatCentsToEuro(cents: number): string`
  - `parseEuroToCents(raw: string): number`
  - `formatBasisPointsToPercent(basisPoints: number): string`
  - `parsePercentToBasisPoints(raw: string): number`
  - `formatIsoDateFr(isoDate: string): string`
  - `formatPeriodMonthFr(periodMonth: string): string`
  - `getCurrentPeriodMonth(now?: Date): string`
  - `class ApiError extends Error { readonly status: number; readonly code?: string; constructor(status: number, message: string, code?: string) }`
  - `interface HttpClient { requestJson<T>(path: string, init?: RequestInit): Promise<T>; requestText(path: string, init?: RequestInit): Promise<string>; requestMultipart<T>(path: string, formData: FormData): Promise<T>; }`
  - `class FetchHttpClient implements HttpClient`

- [ ] **Step 1: Install `mode-watcher`, `vitest`, and `shadcn-svelte` UI components in `web/`**

Run in `web/`:
```bash
npm --prefix web install mode-watcher
npm --prefix web install -D vitest
npx --prefix web shadcn-svelte@latest add -y button card badge table tabs sheet dialog field input label select checkbox toggle-group separator dropdown-menu alert empty skeleton spinner sidebar tooltip sonner
```
Add `"test": "vitest run"` to `scripts` in `web/package.json`. Update `web/vite.config.ts` to add `server.proxy` for `/api`, `/mcp`, and `/healthz` targeting `process.env.PUBLIC_API_ORIGIN ?? 'http://localhost:8080'`, and create `web/src/routes/+layout.ts` exporting `export const ssr = false;`.

- [ ] **Step 2: Write the failing unit tests for money/basis-point formatters and single-flight HTTP client**

Create `web/src/lib/core/formatters/money.test.ts` and `web/src/lib/core/api/http-client.test.ts`:
```ts
// web/src/lib/core/formatters/money.test.ts
import { describe, expect, it } from 'vitest';
import {
	formatBasisPointsToPercent,
	formatCentsToEuro,
	parseEuroToCents,
	parsePercentToBasisPoints
} from './money';

describe('money and basis points boundary formatters', () => {
	it('formats integer cents to French euro strings without float drift', () => {
		expect(formatCentsToEuro(0)).toContain('0,00');
		expect(formatCentsToEuro(92050)).toMatch(/920,50\s*€/);
		expect(formatCentsToEuro(-142000)).toMatch(/-1[\s\u202f\u00a0]?420,00\s*€/);
	});

	it('parses French and dot decimal euro inputs into exact integer cents', () => {
		expect(parseEuroToCents('1 234,56 €')).toBe(123456);
		expect(parseEuroToCents('920,5')).toBe(92050);
		expect(parseEuroToCents('580')).toBe(58000);
		expect(() => parseEuroToCents('-10')).toThrow(/invalid monetary amount/i);
		expect(() => parseEuroToCents('abc')).toThrow(/invalid monetary amount/i);
	});

	it('formats and parses basis points (10000 = 100.00%)', () => {
		expect(formatBasisPointsToPercent(10000)).toMatch(/100,00\s*%/);
		expect(formatBasisPointsToPercent(2000)).toMatch(/20,00\s*%/);
		expect(parsePercentToBasisPoints('20,5 %')).toBe(2050);
	});
});
```

```ts
// web/src/lib/core/api/http-client.test.ts
import { describe, expect, it, vi } from 'vitest';
import { FetchHttpClient } from './http-client';
import { ApiError } from './api-error';

describe('FetchHttpClient', () => {
	it('attaches Bearer token and coalesces concurrent 401 refresh calls into a single flight', async () => {
		let accessToken = 'expired-token';
		let refreshCalls = 0;

		const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			const auth = (init?.headers as Record<string, string>)?.Authorization;
			if (url.endsWith('/api/v1/entities') && auth === 'Bearer expired-token') {
				return new Response(JSON.stringify({ error: 'token expired' }), { status: 401 });
			}
			return new Response(JSON.stringify([{ id: 'ent_1', name: 'SCI Alpha' }]), { status: 200 });
		});

		const client = new FetchHttpClient({
			baseUrl: 'http://localhost:8080',
			fetchFn: fetchMock as unknown as typeof fetch,
			getAccessToken: () => accessToken,
			refreshSession: async () => {
				refreshCalls += 1;
				await new Promise((r) => setTimeout(r, 10));
				accessToken = 'fresh-token';
				return accessToken;
			},
			onUnauthorized: vi.fn()
		});

		const [res1, res2] = await Promise.all([
			client.requestJson<Array<{ id: string }>>('/api/v1/entities'),
			client.requestJson<Array<{ id: string }>>('/api/v1/entities')
		]);

		expect(refreshCalls).toBe(1);
		expect(res1[0].id).toBe('ent_1');
		expect(res2[0].id).toBe('ent_1');
	});

	it('throws ApiError with backend message on non-2xx responses', async () => {
		const client = new FetchHttpClient({
			baseUrl: 'http://localhost:8080',
			fetchFn: vi.fn(async () =>
				new Response(JSON.stringify({ error: 'account is pending approval' }), { status: 403 })
			) as unknown as typeof fetch
		});

		await expect(client.requestJson('/api/v1/auth/login')).rejects.toThrowError(ApiError);
	});
});
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `npm --prefix web test`
Expected: FAIL with module resolution errors for `./money` and `./http-client`.

- [ ] **Step 4: Implement `money.ts`, `date.ts`, `api-error.ts`, and `http-client.ts`**

Implement integer-only string parsing in `web/src/lib/core/formatters/money.ts` (split integer and fractional digits after stripping spaces/`€`, pad fractional part to 2 digits, reject `>2` fractional digits or negative values unless explicitly allowed), French date helpers in `web/src/lib/core/formatters/date.ts`, `ApiError` in `web/src/lib/core/api/api-error.ts`, and `FetchHttpClient` with a shared `refreshPromise: Promise<string | null> | null` in `web/src/lib/core/api/http-client.ts`.

- [ ] **Step 5: Run tests and typecheck**

Run: `npm --prefix web test && npm --prefix web run check`
Expected: PASS with 0 errors.

- [ ] **Step 6: Commit**

```bash
git add web/ docs/superpowers/
git commit -m "feat(web): add shadcn-svelte primitives, formatters, and single-flight HTTP client"
```

---

### Task 2: Domain Modules API Services, Workspace State & Pure Cockpit Builder

**Files:**
- Create: `web/src/lib/core/workspace/workspace.svelte.ts`
- Create: `web/src/lib/modules/auth/types.ts`
- Create: `web/src/lib/modules/auth/api.ts`
- Create: `web/src/lib/modules/auth/auth-store.svelte.ts`
- Create: `web/src/lib/modules/portfolio/types.ts`
- Create: `web/src/lib/modules/portfolio/api.ts`
- Create: `web/src/lib/modules/lease/types.ts`
- Create: `web/src/lib/modules/lease/api.ts`
- Create: `web/src/lib/modules/payment/types.ts`
- Create: `web/src/lib/modules/payment/api.ts`
- Create: `web/src/lib/modules/document/types.ts`
- Create: `web/src/lib/modules/document/api.ts`
- Create: `web/src/lib/modules/inspection/types.ts`
- Create: `web/src/lib/modules/inspection/api.ts`
- Create: `web/src/lib/modules/accounting/types.ts`
- Create: `web/src/lib/modules/accounting/api.ts`
- Create: `web/src/lib/modules/cockpit/types.ts`
- Create: `web/src/lib/modules/cockpit/cockpit-builder.ts`
- Create: `web/src/lib/modules/registry.ts`
- Test: `web/src/lib/modules/cockpit/cockpit-builder.test.ts`

**Interfaces:**
- Consumes: `HttpClient`, `FetchHttpClient`, `getCurrentPeriodMonth` from Task 1.
- Produces:
  - `workspaceStore` (`selectedEntityId: 'ALL' | string`, `selectedPeriodMonth: string`, `setEntityId(id: 'ALL' | string): void`, `setPeriodMonth(month: string): void`)
  - `authStore` (`user: UserDTO | null`, `accessToken: string | null`, `isAuthenticated: boolean`, `isSuperAdmin: boolean`, `login(...)`, `signup(...)`, `logout()`, `initialize()`)
  - `apiRegistry` (`auth`, `portfolio`, `lease`, `payment`, `document`, `inspection`, `accounting`)
  - `buildCockpitEntityGroups(input: CockpitAggregationInput): EntityCockpitGroup[]`

- [ ] **Step 1: Write the failing unit tests for `buildCockpitEntityGroups` in `web/src/lib/modules/cockpit/cockpit-builder.test.ts`**

```ts
// web/src/lib/modules/cockpit/cockpit-builder.test.ts
import { describe, expect, it } from 'vitest';
import { buildCockpitEntityGroups } from './cockpit-builder';

describe('buildCockpitEntityGroups', () => {
	it('groups properties and lots by legal entity and derives prioritized CTAs per row', () => {
		const groups = buildCockpitEntityGroups({
			periodMonth: '2026-10',
			entities: [
				{
					id: 'ent_1',
					name: 'SCI Les Oliviers',
					entity_type: 'SCI_IS',
					siret: '12345678900010',
					naf_code: '6820B',
					address: '10 rue République, Lyon',
					share_capital_cents: 100000,
					total_shares: 100,
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			],
			propertiesByEntity: {
				ent_1: [
					{
						id: 'prop_1',
						legal_entity_id: 'ent_1',
						label: 'Immeuble République',
						property_type: 'APARTMENT_BUILDING',
						accounting_category: 'FIXED_ASSET',
						address: '10 rue République, Lyon',
						acquisition_cents: 25000000,
						work_stock_cents: 0,
						total_book_value_cents: 25000000,
						lots: [
							{
								id: 'lot_1',
								property_id: 'prop_1',
								label: 'Appt T3 République',
								lot_type: 'APARTMENT',
								surface_m2: 65,
								ownership_share: 500,
								floor: '2',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							},
							{
								id: 'lot_2',
								property_id: 'prop_1',
								label: 'Studio Vacant',
								lot_type: 'STUDIO',
								surface_m2: 24,
								ownership_share: 200,
								floor: '1',
								created_at: '2026-01-01T00:00:00Z',
								updated_at: '2026-01-01T00:00:00Z'
							}
						],
						created_at: '2026-01-01T00:00:00Z',
						updated_at: '2026-01-01T00:00:00Z'
					}
				]
			},
			leases: [
				{
					id: 'lease_1',
					legal_entity_id: 'ent_1',
					lot_id: 'lot_1',
					tenant_id: 'ten_1',
					lease_type: 'RESIDENTIAL_EMPTY',
					rent_net_cents: 85000,
					charges_cents: 7000,
					subject_to_vat: false,
					vat_rate_basis_points: 0,
					security_deposit_cents: 85000,
					due_day_of_month: 5,
					gli_policy_number: 'GLI-001',
					guarantor_name: '',
					start_date: '2025-01-01T00:00:00Z',
					active: true,
					created_at: '2025-01-01T00:00:00Z',
					updated_at: '2025-01-01T00:00:00Z'
				}
			],
			tenants: [
				{
					id: 'ten_1',
					full_name: 'Marc Martin',
					email: 'marc@example.com',
					phone: '0600000000',
					address: 'Lyon',
					archived: false,
					created_at: '2025-01-01T00:00:00Z',
					updated_at: '2025-01-01T00:00:00Z'
				}
			],
			rentCallsByEntity: {
				ent_1: [
					{
						id: 'rc_1',
						lease_id: 'lease_1',
						legal_entity_id: 'ent_1',
						tenant_id: 'ten_1',
						period_month: '2026-10',
						rent_net_cents: 85000,
						vat_cents: 0,
						charges_cents: 7000,
						total_due_cents: 92000,
						remaining_due_cents: 92000,
						status: 'PENDING',
						due_date: '2026-10-05T00:00:00Z',
						created_at: '2026-10-01T00:00:00Z',
						updated_at: '2026-10-01T00:00:00Z'
					}
				]
			},
			negativePositionsByEntity: {
				ent_1: [
					{
						lease_id: 'lease_1',
						legal_entity_id: 'ent_1',
						lot_id: 'lot_1',
						tenant_id: 'ten_1',
						tenant_name: 'Marc Martin',
						total_unpaid_cents: 92000,
						unpaid_months: ['2026-10'],
						oldest_due_date: '2026-10-05T00:00:00Z',
						gli_policy_number: 'GLI-001',
						gli_eligible: true
					}
				]
			},
			incidents: [
				{
					id: 'inc_1',
					property_id: 'prop_1',
					lot_id: 'lot_1',
					lease_id: 'lease_1',
					title: 'Fuite ballon ECS',
					status: 'OPEN',
					occurred_at: '2026-10-10',
					estimated_cost_cents: 35000,
					observations: [],
					photo_r2_keys: [],
					created_at: '2026-10-10T00:00:00Z',
					updated_at: '2026-10-10T00:00:00Z'
				}
			]
		});

		expect(groups).toHaveLength(1);
		expect(groups[0].expectedRentCents).toBe(92000);
		expect(groups[0].collectedRentCents).toBe(0);
		expect(groups[0].rows).toHaveLength(2);

		const leasedRow = groups[0].rows[0];
		expect(leasedRow.actions.map((a) => a.kind)).toEqual(['remind', 'pay', 'incident']);
		expect(leasedRow.actions[0].href).toContain('/rents?entityId=ent_1&leaseId=lease_1');

		const vacantRow = groups[0].rows[1];
		expect(vacantRow.isVacant).toBe(true);
		expect(vacantRow.actions[0].kind).toBe('create-lease');
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm --prefix web test`
Expected: FAIL with `./cockpit-builder` not found.

- [ ] **Step 3: Implement DTO types, API services, `workspace.svelte.ts`, `auth-store.svelte.ts`, `cockpit-builder.ts`, and `registry.ts`**

Create each module's `types.ts` and `api.ts` matching the exact Go backend endpoints and JSON tags documented in the spec:
- `AuthApiService`: `POST /api/v1/auth/login`, `POST /api/v1/auth/signup`, `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`, `POST /api/v1/auth/mcp-tokens`, `GET /api/v1/admin/users/pending`, `POST /api/v1/admin/users/{id}/approve`, `POST /api/v1/admin/users/{id}/reject`.
- `PortfolioApiService`: `GET/POST /api/v1/entities`, `GET /api/v1/entities/{id}`, `POST /api/v1/entities/{id}/shareholders`, `GET/POST /api/v1/entities/{id}/properties`, `POST /api/v1/properties/{id}/lots`.
- `LeaseApiService`: `GET/POST /api/v1/tenants`, `POST /api/v1/tenants/{id}/archive`, `GET/POST /api/v1/leases`, `POST /api/v1/entities/{id}/rent-calls/generate`, `GET /api/v1/entities/{id}/rent-calls`.
- `PaymentApiService`: `POST /api/v1/payments/cash`, `POST /api/v1/payments/bank`, `POST /api/v1/payments/cash-deposit`, `POST /api/v1/payments/import-csv`, `GET /api/v1/entities/{id}/negative-positions`, `GET /api/v1/entities/{id}/cash-journal`.
- `DocumentApiService`: `GET/POST /api/v1/document-templates`, `POST /api/v1/documents/generate`, `POST /api/v1/documents/upload`, `POST /api/v1/leases/{id}/gli-dossier`, `GET /api/v1/leases/{id}/documents`.
- `InspectionApiService`: `GET/POST /api/v1/inspection-templates`, `GET /api/v1/inspection-templates/{id}`, `POST /api/v1/inspections`, `GET /api/v1/inspections/{id}`, `PUT /api/v1/inspections/{id}/items`, `POST /api/v1/inspections/{id}/pdf`, `GET/POST /api/v1/incidents`, `GET/PATCH /api/v1/incidents/{id}`.
- `AccountingApiService`: `GET/POST /api/v1/entities/{id}/journal-entries`, `PUT /api/v1/entities/{id}/statements/lines`, `POST /api/v1/entities/{id}/statements/preview`, `POST /api/v1/entities/{id}/statements/pdf`, `GET /api/v1/entities/{id}/fec`.
- `buildCockpitEntityGroups`: Pure function producing `EntityCockpitGroup[]` with ordered `PendingTaskAction[]` (`remind`, `pay`, `generate-calls`, `incident`, `create-lease`).

- [ ] **Step 4: Run tests and typecheck**

Run: `npm --prefix web test && npm --prefix web run check`
Expected: PASS with 0 errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/
git commit -m "feat(web): implement typed module API services, workspace store, and cockpit builder"
```

---

### Task 3: Root Layout (`ModeWatcher`), Authentication Screens (`/login`, `/signup`) & Authenticated Application Shell

**Files:**
- Modify: `web/src/routes/+layout.svelte`
- Delete: `web/src/routes/+page.svelte`
- Create: `web/src/lib/components/layout/ThemeToggle.svelte`
- Create: `web/src/lib/components/layout/EntityWorkspaceSwitcher.svelte`
- Create: `web/src/lib/components/layout/AppSidebar.svelte`
- Create: `web/src/lib/components/layout/PageHeader.svelte`
- Create: `web/src/routes/(auth)/login/+page.svelte`
- Create: `web/src/routes/(auth)/signup/+page.svelte`
- Create: `web/src/routes/(app)/+layout.svelte`

**Interfaces:**
- Consumes: `authStore`, `workspaceStore`, `apiRegistry` from Task 2; `mode-watcher` and `shadcn-svelte` UI components from Task 1.
- Produces:
  - `<ThemeToggle />` using `setMode` / `resetMode` from `"mode-watcher"` inside a `DropdownMenu`.
  - `<EntityWorkspaceSwitcher entities={entities} />` allowing selection of `"ALL"` (*Toutes les entités*) or a specific entity.
  - `<AppSidebar entities={entities} pendingTaskCount={...} openIncidentCount={...} />`
  - `<PageHeader title={...} subtitle={...} actions={snippet} />`

- [ ] **Step 1: Update `web/src/routes/+layout.svelte` with `ModeWatcher` and `Toaster`**

Mount `<ModeWatcher />` from `"mode-watcher"` and `<Toaster />` from `#lib/components/ui/sonner` in `web/src/routes/+layout.svelte`, and call `authStore.initialize()` on mount.

- [ ] **Step 2: Implement `ThemeToggle.svelte`, `EntityWorkspaceSwitcher.svelte`, `AppSidebar.svelte`, and `PageHeader.svelte`**

Follow official `shadcn-svelte` rules:
- `ThemeToggle.svelte`: Uses `DropdownMenu.Root` > `DropdownMenu.Trigger` (`Button` `variant="outline"` `size="icon"` with `@tabler/icons-svelte` `IconSun` / `IconMoon`) and `DropdownMenu.Content` > `DropdownMenu.Group` > `DropdownMenu.Item` calling `setMode("light")`, `setMode("dark")`, `resetMode()`.
- `EntityWorkspaceSwitcher.svelte`: Uses `Select.Root` (`type="single"`) > `Select.Trigger` and `Select.Content` > `Select.Group` > `Select.Item` for `"ALL"` (*Toutes les entités*) and each `LegalEntityDTO`.
- `AppSidebar.svelte`: Uses `Sidebar.*` primitives (`Sidebar.Root`, `Sidebar.Header`, `Sidebar.Content`, `Sidebar.Group`, `Sidebar.Menu`, `Sidebar.MenuItem`, `Sidebar.MenuButton`, `Sidebar.Footer`) with navigation items for `/`, `/rents`, `/inspections`, `/properties`, `/leases`, `/accounting`, `/entities`, and `/admin`.

- [ ] **Step 3: Implement `/login`, `/signup`, and `(app)/+layout.svelte`**

- `web/src/routes/(auth)/login/+page.svelte`: Uses `Card.*` and `Field.FieldGroup` + `Field.Field` + `Input` + `Button` (`Spinner` when submitting). Displays an `Alert.Root` when account is pending approval (`403`) or credentials fail.
- `web/src/routes/(auth)/signup/+page.svelte`: Submits `POST /api/v1/auth/signup` (`202 Accepted`) and displays a French confirmation `Alert.Root` explaining that a `SUPER_ADMIN` must approve the account before login.
- `web/src/routes/(app)/+layout.svelte`: Redirects unauthenticated visitors to `/login`, loads the user's legal entities into `workspaceStore`, and wraps `{@render children()}` in `Sidebar.Provider` + `AppSidebar` + top bar with `Sidebar.Trigger`, active context indicator, and `ThemeToggle`.
- Remove the placeholder `web/src/routes/+page.svelte` so `web/src/routes/(app)/+page.svelte` owns `/`.

- [ ] **Step 4: Verify build and typecheck**

Run: `npm --prefix web run check && npm --prefix web test`
Expected: PASS with 0 errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/
git commit -m "feat(web): add ModeWatcher dark mode, auth screens, and sidebar workspace shell"
```

---

### Task 4: Operational Cockpit (`/`) & Rents, Rent Calls, Payments, Reminders & GLI Dossier (`/rents`)

**Files:**
- Create: `web/src/routes/(app)/+page.svelte`
- Create: `web/src/lib/components/rents/RentActionSheet.svelte`
- Create: `web/src/routes/(app)/rents/+page.svelte`

**Interfaces:**
- Consumes: `buildCockpitEntityGroups`, `apiRegistry`, `workspaceStore`, `formatCentsToEuro`, `parseEuroToCents`, `formatPeriodMonthFr`.
- Produces:
  - `/` (`web/src/routes/(app)/+page.svelte`): Option B Cockpit — Legal Entity cards with monthly collection progress and Property/Lot rows displaying inline CTA buttons (`Relancer impayé →`, `Saisir loyer →`, `Générer appel →`, `Incident ouvert →`, `Créer un bail →`).
  - `/rents` (`web/src/routes/(app)/rents/+page.svelte` + `RentActionSheet.svelte`): Option A Unified Monthly Table + right-hand `Sheet` that opens automatically from Cockpit CTA query params (`?entityId=...&leaseId=...&rentCallId=...&action=pay|remind|generate-calls`) or row click.

- [ ] **Step 1: Implement Operational Cockpit in `web/src/routes/(app)/+page.svelte`**

- Loads entities, properties/lots, leases, tenants, current period rent calls, negative positions, and open incidents (filtered by `workspaceStore.selectedEntityId` when not `"ALL"`).
- Computes `EntityCockpitGroup[]` via `buildCockpitEntityGroups`.
- Renders top summary KPIs (`Encaissé sur la période`, `Reste à encaisser`, `Impayés à relancer`, `Incidents ouverts`), a period month selector (`Input type="month"`), and one `Card.Root` per Legal Entity:
  - Header: Entity name, `Badge` for `entity_type`, collection progress (`formatCentsToEuro(collectedRentCents)` / `formatCentsToEuro(expectedRentCents)`), and entity-level CTA button `Générer appels du mois (N) →` when uncalled active leases exist.
  - Body: `Table.*` of Property/Lot rows showing Property & Lot label, Tenant name, monthly rent + charges, and inline CTA `Button`s on the right side of each row linking directly to the target screen with query params.
  - Uses `Empty.Root` when no entities exist yet, with a CTA to `/entities?action=create`.

- [ ] **Step 2: Implement `RentActionSheet.svelte` and `web/src/routes/(app)/rents/+page.svelte`**

- `web/src/routes/(app)/rents/+page.svelte`:
  - Syncs `entityId` from `$page.url.searchParams` into `workspaceStore.selectedEntityId` if provided.
  - Top toolbar: Month selector (`YYYY-MM`), `ToggleGroup.Root` status filter (`ALL`, `TO_COLLECT`, `OVERDUE`, `PAID`), `Button` to generate monthly rent calls (`POST /api/v1/entities/{id}/rent-calls/generate`), `Dialog` for CSV bank import (`POST /api/v1/payments/import-csv`), and `Dialog` for cash bank deposit (`POST /api/v1/payments/cash-deposit`).
  - Unified `Table.*` of monthly rent calls and active leases, plus collapsible PCG 530 Cash Journal (`GET /api/v1/entities/{id}/cash-journal`).
  - Auto-opens `RentActionSheet` when `leaseId` or `rentCallId` is present in URL query params or when a row is clicked.
- `RentActionSheet.svelte`:
  - Uses `Sheet.Root` + `Sheet.Content` + `Sheet.Header` + `Sheet.Title` + `Sheet.Description`.
  - Section 1 — **Saisir un règlement :** `Field.FieldGroup` with amount in euros (parsed via `parseEuroToCents`), method (`BANK_TRANSFER`, `CHECK`, `DIRECT_DEBIT`, `CASH`), payment date, receipt number / external ref. Calls `recordBankPayment` or `recordCashPayment`.
  - Section 2 — **Avis d'échéance & Quittances :** Generates document from template (`POST /api/v1/documents/generate`) and lists lease documents (`GET /api/v1/leases/{id}/documents`) with download links.
  - Section 3 — **Relances & Dossier Sinistre GLI :** Displays unpaid months from `NegativePositionDTO` and provides a 1-click button to assemble and download the GLI claim `.zip` archive (`POST /api/v1/leases/{id}/gli-dossier`).

- [ ] **Step 3: Run tests and typecheck**

Run: `npm --prefix web test && npm --prefix web run check`
Expected: PASS with 0 errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/
git commit -m "feat(web): implement operational cockpit and unified rents workspace with contextual sheet"
```

---

### Task 5: Incidents & Inspections (`/inspections`) and Portfolio Screens (`/entities`, `/properties`, `/leases`)

**Files:**
- Create: `web/src/routes/(app)/inspections/+page.svelte`
- Create: `web/src/routes/(app)/entities/+page.svelte`
- Create: `web/src/routes/(app)/properties/+page.svelte`
- Create: `web/src/routes/(app)/leases/+page.svelte`

**Interfaces:**
- Consumes: `apiRegistry`, `workspaceStore`, `formatCentsToEuro`, `parseEuroToCents`, `formatBasisPointsToPercent`, `parsePercentToBasisPoints`, `formatIsoDateFr`.
- Produces:
  - `/inspections`: Incidents table + contextual `Sheet` (auto-opened on `?incidentId=...`) and Move-In / Move-Out Inspections (`ENTRANT` / `SORTANT`) with room-by-room condition ratings and comparative PDF generation.
  - `/entities`: Legal Entities list, Entity Portfolio Summary (`total_book_value_cents`), and CERFA 2059-F Shareholders management via `Sheet`.
  - `/properties`: Properties (`FIXED_ASSET` / `MERCHANDISE`) and nested Lots management via `Sheet`.
  - `/leases`: Leases (`RESIDENTIAL_EMPTY`, `RESIDENTIAL_FURNISHED`, `MOBILITY`, `COMMERCIAL_3_6_9`) & Tenants (with GDPR archive and `.docx` contract generation / document upload) via `Sheet`.

- [ ] **Step 1: Implement `/inspections` (`web/src/routes/(app)/inspections/+page.svelte`)**

- Reads `?entityId=...&propertyId=...&incidentId=...&tab=incidents|inspections` from URL search params and auto-opens the right-hand incident `Sheet` when `incidentId` is present.
- `Tabs.Root` with `Tabs.List` (`Incidents & Sinistres`, `États des lieux & Modèles`):
  - **Incidents tab:** `Table.*` of incidents (`OPEN`, `IN_PROGRESS`, `RESOLVED`, `CLOSED`), `Button` to open creation `Sheet` (`POST /api/v1/incidents`), and edit `Sheet` (`PATCH /api/v1/incidents/{id}`) for status, insurance claim number, estimated cost (`parseEuroToCents`), and observations.
  - **Inspections tab:** Create/list inspection templates (`GET/POST /api/v1/inspection-templates`), start Move-In (`ENTRANT`) or Move-Out (`SORTANT`) inspection (`POST /api/v1/inspections`), edit room-by-room conditions (`NEW`, `GOOD`, `FAIR`, `POOR`, `DAMAGED`) in a `Sheet` (`PUT /api/v1/inspections/{id}/items`), and generate/download the comparative PDF (`POST /api/v1/inspections/{id}/pdf`).

- [ ] **Step 2: Implement `/entities`, `/properties`, and `/leases`**

- `web/src/routes/(app)/entities/+page.svelte`: Lists legal entities, loads `GET /api/v1/entities/{id}` summary for the selected entity, displays book value cards (`Acquisition`, `Stock travaux`, `Valeur nette comptable`) and CERFA 2059-F shareholders table (`detention_basis_points` formatted via `formatBasisPointsToPercent`), with right-hand `Sheet`s for creating an entity and upserting a shareholder.
- `web/src/routes/(app)/properties/+page.svelte`: Lists properties and nested lots for the active entity(ies), with right-hand `Sheet`s to create a property (`POST /api/v1/entities/{id}/properties`) and add a lot (`POST /api/v1/properties/{id}/lots`).
- `web/src/routes/(app)/leases/+page.svelte`: `Tabs.Root` for `Baux locatifs` and `Locataires`. Right-hand `Sheet`s to create a lease (`POST /api/v1/leases`), create/archive a tenant (`POST /api/v1/tenants/{id}/archive`), and manage lease templates/documents (`/api/v1/document-templates`, `/api/v1/documents/generate`, `/api/v1/documents/upload`).

- [ ] **Step 3: Run tests and typecheck**

Run: `npm --prefix web test && npm --prefix web run check`
Expected: PASS with 0 errors.

- [ ] **Step 4: Commit**

```bash
git add web/src/routes/\(app\)/
git commit -m "feat(web): implement inspections, incidents, entities, properties, and leases workspaces"
```

---

### Task 6: Interactive Annual Accounts & Tax Package (`/accounting`) and Administration (`/admin`)

**Files:**
- Create: `web/src/lib/modules/accounting/statement-state.ts`
- Create: `web/src/routes/(app)/accounting/+page.svelte`
- Create: `web/src/routes/(app)/admin/+page.svelte`
- Test: `web/src/lib/modules/accounting/statement-state.test.ts`

**Interfaces:**
- Consumes: `apiRegistry`, `authStore`, `workspaceStore`, `formatCentsToEuro`, `parseEuroToCents`, `formatBasisPointsToPercent`.
- Produces:
  - `toggleDisabledLineId(currentIds: string[], lineId: string): string[]`
  - `/accounting` (`web/src/routes/(app)/accounting/+page.svelte`): Interactive financial statements preview (`Bilan`, `Compte de résultat`, `SIG`) with live line toggling, CERFA 2072/2033 PDF generation, normative 18-column DGFiP FEC `.tsv` export, and PCG double-entry journal management.
  - `/admin` (`web/src/routes/(app)/admin/+page.svelte`): Personal MCP token creation and `SUPER_ADMIN` pending user approval/rejection.

- [ ] **Step 1: Write the failing unit test for interactive statement line toggling in `web/src/lib/modules/accounting/statement-state.test.ts`**

```ts
// web/src/lib/modules/accounting/statement-state.test.ts
import { describe, expect, it } from 'vitest';
import { toggleDisabledLineId } from './statement-state';

describe('toggleDisabledLineId', () => {
	it('adds and removes line IDs deterministically without duplicates', () => {
		const step1 = toggleDisabledLineId([], 'acct_681100');
		expect(step1).toEqual(['acct_681100']);

		const step2 = toggleDisabledLineId(step1, 'acct_615000');
		expect(step2).toEqual(['acct_681100', 'acct_615000']);

		const step3 = toggleDisabledLineId(step2, 'acct_681100');
		expect(step3).toEqual(['acct_615000']);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm --prefix web test`
Expected: FAIL with `./statement-state` not found.

- [ ] **Step 3: Implement `statement-state.ts`, `/accounting`, and `/admin`**

- `web/src/lib/modules/accounting/statement-state.ts`: Pure helper `toggleDisabledLineId` and section grouping helpers (`OPERATING_REVENUE`, `EXTERNAL_EXPENSES`, `DEPRECIATION`, `FIXED_ASSETS`, `CURRENT_ASSETS`, `EQUITY`, `LIABILITIES`, etc.) with French PCG/CERFA display labels.
- `web/src/routes/(app)/accounting/+page.svelte`:
  - Fiscal year selector (`default: current year`) and active entity selector.
  - Tab 1 — **Comptes Annuels Interactifs & Liasse 2072/2033 :** Summary KPI cards (`Total Produits`, `Total Charges`, `Résultat Net`, `CAF`, `Total Actif`, `Total Capitaux Propres & Passif`), interactive table of `ComputedLineDTO` rows with `Checkbox` (`shadcn-svelte`) bound to `toggleDisabledLineId` triggering live `previewStatements` recomputation, `Button` to generate Annual Accounts PDF (`POST /api/v1/entities/{id}/statements/pdf`), `Button` to download DGFiP FEC `.tsv` (`GET /api/v1/entities/{id}/fec`), and `Sheet` to upsert balance lines (`PUT /api/v1/entities/{id}/statements/lines`).
  - Tab 2 — **Journal Comptable PCG :** Table of `JournalEntryDTO` entries and right-hand `Sheet` to record balanced double-entry PCG journal entries (`POST /api/v1/entities/{id}/journal-entries`) with live debit == credit check.
- `web/src/routes/(app)/admin/+page.svelte`:
  - Card 1 — **Jetons MCP (Claude Desktop / Agents IA) :** Form to create a labeled MCP token (`POST /api/v1/auth/mcp-tokens`) and display the one-time token in an `Alert.Root` with copy button.
  - Card 2 (`{#if authStore.isSuperAdmin}`) — **Approbation des comptes en attente :** Table of `GET /api/v1/admin/users/pending` with `Approuver` (`POST /api/v1/admin/users/{id}/approve`) and `Rejeter` (`POST /api/v1/admin/users/{id}/reject`) buttons.

- [ ] **Step 4: Run full frontend test suite, typecheck, and production build**

Run: `npm --prefix web test && npm --prefix web run check && npm --prefix web run build`
Expected: All unit tests pass, `svelte-check` reports 0 errors and 0 warnings, and Vite production build succeeds.

- [ ] **Step 5: Commit**

```bash
git add web/
git commit -m "feat(web): implement interactive annual accounts, FEC export, and admin governance"
```
