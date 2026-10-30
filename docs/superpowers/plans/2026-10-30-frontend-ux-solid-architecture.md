# Frontend UX, Sidebar-07 Navigation, Persistent Onboarding & SOLID Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the SvelteKit 2 / Svelte 5 frontend (`web/`) into a `sidebar-07` navigation backed by dedicated nested routes, a persistent first-run Legal Entity creation modal, centered `Dialog` modals for all 15 resource creation/edit forms (keeping `Sheet` drawers exclusively for the 3 multi-action detail workspaces), and small SOLID components without modifying `web/src/lib/components/ui/**`.

**Architecture:** Extract pure TypeScript domain-label dictionaries (`#lib/modules/*/labels.ts`), onboarding-gate logic (`#lib/core/workspace/onboarding-gate.ts`), and navigation state helpers (`#lib/components/layout/sidebar-nav.ts`) tested with Vitest. Split multi-tab route monoliths into single-responsibility SvelteKit nested routes (`/leases/*`, `/rents/*`, `/inspections/*`, `/accounting/*`) first so `resolve('/(app)/...')` in the `sidebar-07` `AppSidebar` compiles with strict SvelteKit `RouteId` typing at every commit. Place components reused across 2+ routes in `web/src/lib/components/` and single-route components in colocated `(components)/` folders.

**Tech Stack:** SvelteKit 2, Svelte 5 (`$props`, `$state`, `$derived`), TypeScript 6, Tailwind CSS v4, `shadcn-svelte` (`bits-ui`, `@tabler/icons-svelte`, `svelte-sonner`), `openapi-fetch`, Vitest.

**Spec:** `docs/superpowers/specs/2026-10-30-frontend-ux-solid-architecture-design.md`

## Global Constraints

- **100% English Code (`AGENTS.md` Rule 1):** All filenames, route directories, component names, prop names, variables, constants, functions, and types MUST be 100% English. French is confined strictly to rendered UI templates and `#lib/modules/*/labels.ts` display strings.
- **Integer Money & Percentages (`AGENTS.md` Rule 2):** Never transport or compute money or rates in floats. Use integer cents/basis points and `#lib/core/formatters/money`.
- **Immutable `web/src/lib/components/ui/**`:** Only run `npx --prefix web shadcn-svelte@latest add collapsible avatar --yes` to add missing official primitives. Never edit existing or newly generated files inside `web/src/lib/components/ui/**`; apply `14px` (`text-sm`), `20px` Tabler icon (`[&_svg]:size-5`), and `12px` spacing (`gap-3`) overrides via consumer-side `class="..."` props.
- **Dedicated Nested Routes Only:** Never use `?view=` or `?tab=` query parameters to switch views. Reserve query parameters strictly for shared filter/pre-fill state (`?year=`, `?entityId=`, `?propertyId=`, `?lotId=`, `?leaseId=`, `?rentCallId=`, `?incidentId=`, `?action=`).
- **Icons:** Strictly `@tabler/icons-svelte` (matching `web/components.json`).
- **SOLID / ISP Components:** Never pass `PageData` or `ActionData` mega-objects into child components; pass narrow typed props and explicit callback props. Each `*Dialog.svelte` encapsulates its own local form `$state`, loading flag, `use:enhance` result handling, toast notification, `invalidateAll()` / `update()`, and reset on success.
- **Mandatory Immediate Formatting (`AGENTS.md` Rule 5):** Run `npm --prefix web run format` immediately after creating or modifying any `.ts`, `.svelte`, or `.css` file in `web/`.

## Review Focus

1. **Backend `listEntities()` outage vs. zero-entity first run:** When `listEntities()` throws a non-`401` error in `(app)/+layout.server.ts`, `entities` is `[]` and `entitiesLoadError` is non-null — the layout MUST show an error `Alert` with retry/logout and MUST NOT lock the user into the persistent `CreateEntityDialog` (pinned in Task 2 `onboarding-gate.test.ts` & `+layout.svelte`).
2. **Cross-route `CreateEntityDialog` submission from non-`/entities` routes (`/`, `/properties`, `/leases`):** Submitting to `/entities?/createEntity` from another route does not populate that route's `form` prop; `entities/+page.server.ts` must persist `gi_entity_id` via `setWorkspaceCookies` and `CreateEntityDialog` must inspect `use:enhance`'s `result` directly, calling `workspaceStore.setEntityId(created.id)` so the new entity becomes active immediately (pinned in Task 2).
3. **Stale OpenAPI enum values in labels, incident filters, and document template pre-selection:** OpenAPI enums (`FAMILY_SARL`, `SOLE_PROPRIETORSHIP`, `JOINT_OWNERSHIP`, `INSURANCE_CLAIM`, `WORK_IN_PROGRESS`, `RENT_RECEIPT`, `FORMAL_NOTICE`, `GLI_DECLARATION`) must map accurately in `#lib/modules/*/labels.ts`, count as open incidents (`status !== 'CLOSED'`), and deep-link Cockpit incidents to `/inspections/incidents?...` without `&tab=` (pinned in Task 1 `labels.test.ts` & `cockpit-builder.test.ts`, applied in Task 5 `RentActionSheet.svelte`).
4. **Malformed or partial euro keystrokes in double-entry journal live balance check:** Typing `"1 200,"` or empty strings in `CreateJournalEntryDialog` must never throw an uncaught exception in `$derived` and must report `{ debitCents, creditCents, balanced: false }` until both parse to equal positive integer cents (pinned in Task 7 `statement-state.test.ts`).
5. **Sidebar active-link collision between parent route and sub-routes (`/leases` vs. `/leases/tenants`):** Collapsible parent groups must open on prefix match (`/leases` or `/leases/*`), whereas sub-items (`Baux actifs` at `/leases` vs. `Locataires` at `/leases/tenants`) must highlight strictly on exact normalized pathname match so two sub-items never highlight at once (pinned in Task 8 `sidebar-nav.test.ts`).

---

### Task 1: Shared Domain Label Dictionaries & Cockpit Deep-Link Alignment

**Files:**
- Create: `web/src/lib/modules/portfolio/labels.ts`
- Create: `web/src/lib/modules/lease/labels.ts`
- Create: `web/src/lib/modules/document/labels.ts`
- Create: `web/src/lib/modules/payment/labels.ts`
- Create: `web/src/lib/modules/inspection/labels.ts`
- Create: `web/src/lib/modules/labels.test.ts`
- Modify: `web/src/lib/modules/cockpit/cockpit-builder.ts`
- Modify: `web/src/lib/modules/cockpit/cockpit-builder.test.ts`

**Interfaces:**
- Produces:
  - `#lib/modules/portfolio/labels`: `ENTITY_TYPE_OPTIONS: ReadonlyArray<{ value: EntityType; label: string }>`, `formatEntityTypeLabel(value: string): string`, `PROPERTY_TYPE_OPTIONS`, `formatPropertyTypeLabel(value: string): string`, `ACCOUNTING_CATEGORY_OPTIONS: ReadonlyArray<{ value: AccountingCategory; label: string }>`, `LOT_TYPE_OPTIONS`, `formatLotTypeLabel(value: string): string`
  - `#lib/modules/lease/labels`: `LEASE_TYPE_OPTIONS: ReadonlyArray<{ value: LeaseType; label: string }>`, `formatLeaseTypeLabel(value: string): string`
  - `#lib/modules/document/labels`: `TEMPLATE_TYPE_OPTIONS: ReadonlyArray<{ value: TemplateType; label: string }>`, `formatTemplateTypeLabel(value: string): string`, `TEMPLATE_FORMAT_OPTIONS: ReadonlyArray<{ value: TemplateFormat; label: string }>`
  - `#lib/modules/payment/labels`: `PAYMENT_METHOD_OPTIONS: ReadonlyArray<{ value: PaymentMethod; label: string }>`, `formatPaymentMethodLabel(value: string): string`
  - `#lib/modules/inspection/labels`: `INCIDENT_STATUS_OPTIONS: ReadonlyArray<{ value: IncidentStatus; label: string }>`, `formatIncidentStatusLabel(status: IncidentStatus): string`, `formatIncidentStatusVariant(status: IncidentStatus): 'destructive' | 'default' | 'secondary' | 'outline'`, `INSPECTION_TYPE_OPTIONS: ReadonlyArray<{ value: InspectionType; label: string }>`, `formatInspectionTypeLabel(type: InspectionType): string`, `CONDITION_STATE_OPTIONS: ReadonlyArray<{ value: ConditionState; label: string }>`, `formatConditionStateLabel(state: ConditionState): string`

- [ ] **Step 1: Write failing unit tests in `web/src/lib/modules/labels.test.ts` and `web/src/lib/modules/cockpit/cockpit-builder.test.ts`**

Assert in `labels.test.ts`:
- Every OpenAPI `EntityType` (`'SCI_IS'`, `'SCI_IR'`, `'SARL_IS'`, `'FAMILY_SARL'`, `'SAS'`, `'SOLE_PROPRIETORSHIP'`, `'JOINT_OWNERSHIP'`) resolves to a distinct French label (never returning the raw enum code).
- Every `LeaseType` (`'RESIDENTIAL_UNFURNISHED'`, `'RESIDENTIAL_FURNISHED'`, `'COMMERCIAL_3_6_9'`, `'PROFESSIONAL'`, `'SHORT_TERM'`, `'PARKING'`), `TemplateType` (`'LEASE'`, `'RENT_RECEIPT'`, `'FORMAL_NOTICE'`, `'GLI_DECLARATION'`, `'CUSTOM'`), `PaymentMethod`, `IncidentStatus`, `InspectionType`, and `ConditionState` maps to its French label.

Add a test in `cockpit-builder.test.ts`:
- Incidents with status `'INSURANCE_CLAIM'` and `'WORK_IN_PROGRESS'` are counted in `openIncidentsCount` (while `'CLOSED'` is excluded), and incident action `href`s point to `/inspections/incidents?entityId=...&propertyId=...&incidentId=...` (containing no `tab=` query parameter).

- [ ] **Step 2: Run Vitest to confirm failure**

Run: `npm --prefix web test`
Expected: FAIL (`Cannot find module './portfolio/labels'` and cockpit incident URL / status assertion failure).

- [ ] **Step 3: Implement the 5 `labels.ts` modules and update `cockpit-builder.ts`**

In `cockpit-builder.ts`:
- Filter open incidents via `incident.status !== 'CLOSED'`.
- Replace `/inspections?...&tab=incidents` with `/inspections/incidents?entityId=${encodeURIComponent(entity.id)}&propertyId=${encodeURIComponent(property.id)}&incidentId=${encodeURIComponent(firstIncidentId)}`.
- Run `npm --prefix web run format` immediately after writing the `.ts` files.

- [ ] **Step 4: Run tests and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS (0 errors).

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/modules
git commit -m "feat(web): centralize domain enum French labels and align cockpit incident deep-links"
```

---

### Task 2: Official `shadcn-svelte` Primitives (`collapsible`, `avatar`), Shared `CreateEntityDialog` & Persistent Onboarding Gate

**Files:**
- Create (via CLI only): `web/src/lib/components/ui/collapsible/*`, `web/src/lib/components/ui/avatar/*`
- Create: `web/src/lib/core/workspace/onboarding-gate.ts`
- Create: `web/src/lib/core/workspace/onboarding-gate.test.ts`
- Create: `web/src/lib/core/workspace/entity-dialog-context.ts`
- Create: `web/src/lib/components/portfolio/CreateEntityDialog.svelte`
- Modify: `web/src/routes/(app)/+layout.server.ts`
- Modify: `web/src/routes/(app)/+layout.svelte`
- Modify: `web/src/routes/(app)/entities/+page.server.ts`

**Interfaces:**
- Consumes: Task 1 `ENTITY_TYPE_OPTIONS`.
- Produces:
  - `deriveWorkspaceGateState(params: { entitiesCount: number; entitiesLoadError: string | null }): 'onboarding_required' | 'load_error' | 'ready'` in `#lib/core/workspace/onboarding-gate`
  - `setEntityDialogContext(api: { openCreateDialog: () => void }): void` and `useEntityDialogContext(): { openCreateDialog: () => void }` in `#lib/core/workspace/entity-dialog-context`
  - `CreateEntityDialog.svelte` props: `{ open?: boolean; persistent?: boolean; onCreated?: (entity: LegalEntityDTO) => void }` (`open = $bindable(false)`)
  - `+layout.server.ts` load output: `{ user: UserDTO; entities: LegalEntityDTO[]; entitiesLoadError: string | null; selectedEntityId: string; selectedPeriodMonth: string }`

- [ ] **Step 1: Install official `collapsible` and `avatar` primitives via `shadcn-svelte` CLI**

Run: `npx --prefix web shadcn-svelte@latest add collapsible avatar --yes && npm --prefix web run format`
Verify: `git status -s web/src/lib/components/ui` shows ONLY untracked `web/src/lib/components/ui/avatar/` and `web/src/lib/components/ui/collapsible/` (zero modified files in `ui/`).

- [ ] **Step 2: Write failing tests in `web/src/lib/core/workspace/onboarding-gate.test.ts`**

Assert:
- `{ entitiesCount: 0, entitiesLoadError: null }` → `'onboarding_required'`
- `{ entitiesCount: 0, entitiesLoadError: 'Backend 502' }` → `'load_error'` (never locks user into onboarding modal!)
- `{ entitiesCount: 2, entitiesLoadError: null }` → `'ready'`
- `{ entitiesCount: 1, entitiesLoadError: 'stale warning' }` → `'ready'`

Run `npm --prefix web test` to confirm FAIL, then implement `onboarding-gate.ts` and `entity-dialog-context.ts` (using Svelte `setContext`/`getContext` with a `Symbol('entity-dialog-context')`), run `npm --prefix web run format`, and confirm Vitest PASS.

- [ ] **Step 3: Update `entities/+page.server.ts` `actions.createEntity` & build `CreateEntityDialog.svelte`**

- In `entities/+page.server.ts` `actions.createEntity`: accept `{ request, locals, cookies, url }` and call `setWorkspaceCookies(cookies, { selectedEntityId: created.id }, url.protocol === 'https:')` upon creation success.
- Create `web/src/lib/components/portfolio/CreateEntityDialog.svelte`:
  - Centered `Dialog.Root` (`open = $bindable(false)`, `onOpenChange={(next) => { if (!persistent) open = next; }}`).
  - `Dialog.Content` (`class="max-h-[90vh] overflow-y-auto sm:max-w-lg"`, `showCloseButton={!persistent}`, `escapeKeydownBehavior={persistent ? 'ignore' : 'close'}`, `interactOutsideBehavior={persistent ? 'ignore' : 'close'}`).
  - Form posts to `action="{resolve('/(app)/entities')}?/createEntity"` via `use:enhance`, inspecting `result`:
    - `result.type === 'success'`: extract `createdEntity` & `successMessage`, show `toast.success`, reset fields, call `workspaceStore.setEntityId(createdEntity.id)` (which triggers `invalidateAll()`), call `onCreated?.(createdEntity)`, and if `!persistent` set `open = false`.
    - `result.type === 'failure'` / `'error'`: display inline `Alert` + `toast.error`.
  - When `persistent` is `true`, hide *Annuler* and display *« Bienvenue — Créez votre première entité juridique »* + *« Créer l'entité juridique et démarrer »*.

- [ ] **Step 4: Wire `(app)/+layout.server.ts` and `(app)/+layout.svelte` onboarding gate**

- In `+layout.server.ts`: fetch only `locals.api.portfolio.listEntities()` (drop `listIncidents()`), returning `entitiesLoadError: null` on success or error message on non-`401` catch.
- In `+layout.svelte`: call `setEntityDialogContext({ openCreateDialog: () => (createEntityDialogOpen = true) })` and render based on `deriveWorkspaceGateState`:
  - `'load_error'`: top bar (`ThemeToggle` + logout form) + destructive `Alert` with *Réessayer* (`invalidateAll()`).
  - `'onboarding_required'`: minimal top bar (`ThemeToggle` + *Se déconnecter* form) + `<CreateEntityDialog open={true} persistent={true} />` (no `{@render children()}`).
  - `'ready'`: `Sidebar.Provider` + `AppSidebar` + header + `{@render children()}` + `<CreateEntityDialog bind:open={createEntityDialogOpen} persistent={false} />`.

- [ ] **Step 5: Format, lint, test, and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/components/ui/avatar web/src/lib/components/ui/collapsible web/src/lib/core/workspace web/src/lib/components/portfolio web/src/routes/\(app\)/+layout.server.ts web/src/routes/\(app\)/+layout.svelte web/src/routes/\(app\)/entities/+page.server.ts
git commit -m "feat(web): add persistent legal entity onboarding gate and shared CreateEntityDialog"
```

---

### Task 3: SOLID Decomposition & Centered Modals for Cockpit (`/`), Entities (`/entities`), Properties (`/properties`) & Admin (`/admin`)

**Files:**
- Create: `web/src/routes/(app)/(components)/CockpitMetricsGrid.svelte`
- Create: `web/src/routes/(app)/(components)/PortfolioCockpitGroupCard.svelte`
- Modify: `web/src/routes/(app)/+page.svelte`
- Create: `web/src/routes/(app)/entities/(components)/EntitiesMetricsGrid.svelte`
- Create: `web/src/routes/(app)/entities/(components)/EntitiesTableCard.svelte`
- Create: `web/src/routes/(app)/entities/(components)/ShareholdersTableCard.svelte`
- Create: `web/src/routes/(app)/entities/(components)/UpsertShareholderDialog.svelte`
- Modify: `web/src/routes/(app)/entities/+page.svelte`
- Create: `web/src/routes/(app)/properties/(components)/PropertyLotsCard.svelte`
- Create: `web/src/routes/(app)/properties/(components)/CreatePropertyDialog.svelte`
- Create: `web/src/routes/(app)/properties/(components)/CreateLotDialog.svelte`
- Modify: `web/src/routes/(app)/properties/+page.svelte`
- Create: `web/src/routes/(app)/admin/(components)/McpTokensCard.svelte`
- Create: `web/src/routes/(app)/admin/(components)/PendingUsersGovernanceCard.svelte`
- Modify: `web/src/routes/(app)/admin/+page.svelte`

**Interfaces:**
- Consumes: Task 1 `#lib/modules/portfolio/labels`; Task 2 `useEntityDialogContext`.
- Produces:
  - `UpsertShareholderDialog.svelte`: `{ open?: boolean; entities: LegalEntityDTO[]; defaultEntityId?: string }` (`open = $bindable(false)`, centered `Dialog`, posts to `?/upsertShareholder`)
  - `CreatePropertyDialog.svelte`: `{ open?: boolean; entities: LegalEntityDTO[]; defaultEntityId?: string; onCreated?: (property: PropertyDTO) => void }` (`open = $bindable(false)`, centered `Dialog`, `sm:max-w-lg`, posts to `?/createProperty`)
  - `CreateLotDialog.svelte`: `{ open?: boolean; properties: PropertyDTO[]; defaultPropertyId?: string }` (`open = $bindable(false)`, centered `Dialog`, posts to `?/createLot`)

- [ ] **Step 1: Decompose Cockpit (`web/src/routes/(app)/+page.svelte`)**

Extract `CockpitMetricsGrid.svelte` and `PortfolioCockpitGroupCard.svelte` (using `formatEntityTypeLabel` & `formatLotTypeLabel` from `#lib/modules/portfolio/labels`). Keep `(app)/+page.svelte` as a slim orchestrator (< 110 lines).

- [ ] **Step 2: Decompose Entities (`web/src/routes/(app)/entities/+page.svelte`) & convert Shareholder `Sheet` to `Dialog`**

Extract `EntitiesMetricsGrid.svelte`, `EntitiesTableCard.svelte` (`{ summaries, selectedEntityId, onSelectEntity }`), `ShareholdersTableCard.svelte` (`{ summary, onAddShareholder }`), and `UpsertShareholderDialog.svelte` (centered `Dialog.Root` + `Dialog.Content class="sm:max-w-md"` with self-contained `$state` and `use:enhance`). Wire *Nouvelle entité* button (and `?action=create`) in `entities/+page.svelte` to `useEntityDialogContext().openCreateDialog()`. Zero `Sheet` imports remain in `/entities`.

- [ ] **Step 3: Decompose Properties (`web/src/routes/(app)/properties/+page.svelte`) & convert Property/Lot `Sheet`s to `Dialog`s**

Extract `PropertyLotsCard.svelte` (`{ property, entityName, activeLeaseByLotId, onAddLot }`), `CreatePropertyDialog.svelte` (centered `Dialog`, `sm:max-w-lg max-h-[90vh] overflow-y-auto`), and `CreateLotDialog.svelte` (centered `Dialog`, `sm:max-w-md`). Preserve `?propertyId=<id>` deep-link pre-selection from Cockpit to open `CreateLotDialog` for that property, and update vacant lot *Créer un bail →* link to `/leases?entityId=...&lotId=...`. Zero `Sheet` imports remain in `/properties`.

- [ ] **Step 4: Decompose Admin (`web/src/routes/(app)/admin/+page.svelte`)**

Extract `McpTokensCard.svelte` (owning its own `mcpLabel`, `isCreating`, `createdToken` `$state` via `use:enhance` result inspection) and `PendingUsersGovernanceCard.svelte` (`{ isSuperAdmin, userRole, pendingUsers, pendingError }`).

- [ ] **Step 5: Format, lint, test, and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/routes/\(app\)/\(components\) web/src/routes/\(app\)/+page.svelte web/src/routes/\(app\)/entities web/src/routes/\(app\)/properties web/src/routes/\(app\)/admin
git commit -m "refactor(web): decompose cockpit, entities, properties, and admin into SOLID components with centered dialogs"
```

---

### Task 4: Leases Bounded Context Nested Routes (`/leases`, `/leases/tenants`, `/leases/templates`)

**Files:**
- Create: `web/src/routes/(app)/leases/(components)/LeasesTableCard.svelte`
- Create: `web/src/routes/(app)/leases/(components)/CreateLeaseDialog.svelte`
- Create: `web/src/routes/(app)/leases/(components)/LeaseDocumentsSheet.svelte`
- Modify: `web/src/routes/(app)/leases/+page.server.ts`
- Modify: `web/src/routes/(app)/leases/+page.svelte`
- Create: `web/src/routes/(app)/leases/tenants/+page.server.ts`
- Create: `web/src/routes/(app)/leases/tenants/+page.svelte`
- Create: `web/src/routes/(app)/leases/tenants/(components)/TenantsTableCard.svelte`
- Create: `web/src/routes/(app)/leases/tenants/(components)/CreateTenantDialog.svelte`
- Create: `web/src/routes/(app)/leases/templates/+page.server.ts`
- Create: `web/src/routes/(app)/leases/templates/+page.svelte`
- Create: `web/src/routes/(app)/leases/templates/(components)/DocumentTemplatesCard.svelte`
- Create: `web/src/routes/(app)/leases/templates/(components)/RegisterTemplateDialog.svelte`

**Interfaces:**
- Consumes: Task 1 `#lib/modules/lease/labels`, `#lib/modules/document/labels`, `#lib/modules/portfolio/labels`.
- Produces:
  - `/leases` (`+page.server.ts`): `load` returns `{ entities, properties, leases, tenants, templates, documentsByLeaseId, errorMessage }`; `actions`: `{ createLease, generateDoc, uploadFile }`.
  - `/leases/tenants` (`+page.server.ts`): `load` returns `{ tenants, activeLeaseCountByTenantId, errorMessage }`; `actions`: `{ createTenant, archiveTenant }`.
  - `/leases/templates` (`+page.server.ts`): `load` returns `{ templates, errorMessage }`; `actions`: `{ registerTemplate }`.

- [ ] **Step 1: Build `/leases/tenants` route & components (`TenantsTableCard.svelte`, `CreateTenantDialog.svelte`)**

Move `createTenant` and `archiveTenant` actions into `leases/tenants/+page.server.ts` (loading `listTenants(true)` and `listLeases('ALL')` to compute `activeLeaseCountByTenantId`). Implement `CreateTenantDialog.svelte` as a centered `Dialog` (`sm:max-w-md`) and `TenantsTableCard.svelte` for directory + GDPR archive form.

- [ ] **Step 2: Build `/leases/templates` route & components (`DocumentTemplatesCard.svelte`, `RegisterTemplateDialog.svelte`)**

Move `registerTemplate` action into `leases/templates/+page.server.ts` (loading only `locals.api.document.listTemplates()`). Implement `RegisterTemplateDialog.svelte` (`Dialog`, `.docx` base64 reader + HTML textarea) and `DocumentTemplatesCard.svelte`.

- [ ] **Step 3: Refactor `/leases` route & components (`LeasesTableCard.svelte`, `CreateLeaseDialog.svelte`, `LeaseDocumentsSheet.svelte`)**

- Trim `leases/+page.server.ts` actions to `{ createLease, generateDoc, uploadFile }`.
- Convert Lease creation from `Sheet` to centered `CreateLeaseDialog.svelte` (`sm:max-w-lg max-h-[90vh] overflow-y-auto`), accepting `{ open = $bindable(false), entities, properties, tenants, preselectedEntityId, preselectedLotId }`. If `tenants.length === 0`, display an inline notice with link to `resolve('/(app)/leases/tenants')`.
- Keep `LeaseDocumentsSheet.svelte` as `Sheet` (detail workspace #1) encapsulating template generation, R2 upload, and archived document list for `selectedLease`.
- In `leases/+page.svelte` (< 130 lines), remove `Tabs`; auto-open `CreateLeaseDialog` when `?lotId=` is present in URL search params.

- [ ] **Step 4: Format, lint, test, and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/routes/\(app\)/leases
git commit -m "feat(web): split leases, tenants, and document templates into nested SOLID routes with centered modals"
```

---

### Task 5: Rents & Cash Ledger Nested Routes (`/rents`, `/rents/cash-journal`)

**Files:**
- Delete: `web/src/lib/components/rents/RentActionSheet.svelte`
- Create: `web/src/routes/(app)/rents/(components)/RentsMetricsGrid.svelte`
- Create: `web/src/routes/(app)/rents/(components)/RentCallsTableCard.svelte`
- Create: `web/src/routes/(app)/rents/(components)/ImportPaymentsCsvDialog.svelte`
- Create: `web/src/routes/(app)/rents/(components)/RentActionSheet.svelte`
- Modify: `web/src/routes/(app)/rents/+page.server.ts`
- Modify: `web/src/routes/(app)/rents/+page.svelte`
- Create: `web/src/routes/(app)/rents/cash-journal/+page.server.ts`
- Create: `web/src/routes/(app)/rents/cash-journal/+page.svelte`
- Create: `web/src/routes/(app)/rents/cash-journal/(components)/CashJournalTableCard.svelte`
- Create: `web/src/routes/(app)/rents/cash-journal/(components)/CashDepositDialog.svelte`

**Interfaces:**
- Consumes: Task 1 `#lib/modules/payment/labels`, `#lib/modules/document/labels`.
- Produces:
  - `/rents` (`+page.server.ts`): `load` returns `{ entities, leases, tenants, templates, propertiesByEntity, rentCallsByEntity, negativePositionsByEntity, documentsByLeaseId, errorMessage }`; `actions`: `{ generateCalls, importCsv, recordPayment, generateDoc, assembleGli }`.
  - `/rents/cash-journal` (`+page.server.ts`): `load` returns `{ entities, cashEntries, errorMessage }`; `actions`: `{ cashDeposit }`.

- [ ] **Step 1: Build `/rents/cash-journal` route & components (`CashJournalTableCard.svelte`, `CashDepositDialog.svelte`)**

Move `cashDeposit` action and `listCashJournal` loading into `rents/cash-journal/+page.server.ts` (aggregating entries across `targetEntities`, sorted newest-first). Extract `CashDepositDialog.svelte` (centered `Dialog`, `sm:max-w-md`) and `CashJournalTableCard.svelte`.

- [ ] **Step 2: Relocate & fix `RentActionSheet.svelte` and decompose `/rents`**

- Create `web/src/routes/(app)/rents/(components)/RentActionSheet.svelte` (and delete `web/src/lib/components/rents/RentActionSheet.svelte`):
  - Fix stale `TemplateType` lookups to exact OpenAPI enum values: `'GLI_DECLARATION'` / `'FORMAL_NOTICE'` for GLI/reminders and `'RENT_RECEIPT'` for receipts (using `PAYMENT_METHOD_OPTIONS` from `#lib/modules/payment/labels`).
  - Handle `use:enhance` results cleanly inside `RentActionSheet.svelte` (storing `assembledGliArchive` locally from `result.data` + `toast.success` / `toast.error` + `invalidateAll()`).
- Extract `RentsMetricsGrid.svelte`, `RentCallsTableCard.svelte`, and `ImportPaymentsCsvDialog.svelte` (`Dialog`).
- Remove `cashDeposit` and `cashJournalByEntity` from `rents/+page.server.ts` & `rents/+page.svelte`; preserve `?leaseId=` & `?action=pay|remind|generate-calls` deep-link handling from Cockpit.

- [ ] **Step 3: Format, lint, test, and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/components/rents web/src/routes/\(app\)/rents
git commit -m "feat(web): split rents and PCG 530 cash journal into nested SOLID routes and fix template enum lookups"
```

---

### Task 6: Inspections & Incidents Nested Routes (`/inspections`, `/inspections/incidents`, `/inspections/templates`)

**Files:**
- Create: `web/src/routes/(app)/inspections/(components)/InspectionsListCard.svelte`
- Create: `web/src/routes/(app)/inspections/(components)/CreateInspectionDialog.svelte`
- Create: `web/src/routes/(app)/inspections/(components)/InspectionGridSheet.svelte`
- Modify: `web/src/routes/(app)/inspections/+page.server.ts`
- Modify: `web/src/routes/(app)/inspections/+page.svelte`
- Create: `web/src/routes/(app)/inspections/incidents/+page.server.ts`
- Create: `web/src/routes/(app)/inspections/incidents/+page.svelte`
- Create: `web/src/routes/(app)/inspections/incidents/(components)/IncidentsTableCard.svelte`
- Create: `web/src/routes/(app)/inspections/incidents/(components)/CreateIncidentDialog.svelte`
- Create: `web/src/routes/(app)/inspections/incidents/(components)/EditIncidentDialog.svelte`
- Create: `web/src/routes/(app)/inspections/templates/+page.server.ts`
- Create: `web/src/routes/(app)/inspections/templates/+page.svelte`
- Create: `web/src/routes/(app)/inspections/templates/(components)/InspectionTemplatesCard.svelte`
- Create: `web/src/routes/(app)/inspections/templates/(components)/CreateInspectionTemplateDialog.svelte`

**Interfaces:**
- Consumes: Task 1 `#lib/modules/inspection/labels`.
- Produces:
  - `/inspections` (`+page.server.ts`): `load` returns `{ inspections, templates, leases, tenants, errorMessage }`; `actions`: `{ createInspection, saveInspectionItems, generatePdf }`.
  - `/inspections/incidents` (`+page.server.ts`): `load` returns `{ incidents, properties, leases, tenants, errorMessage }`; `actions`: `{ createIncident, updateIncident }`.
  - `/inspections/templates` (`+page.server.ts`): `load` returns `{ templates, errorMessage }`; `actions`: `{ createTemplate }`.

- [ ] **Step 1: Build `/inspections/templates` route & components (`InspectionTemplatesCard.svelte`, `CreateInspectionTemplateDialog.svelte`)**

Load `locals.api.inspection.listTemplates()` and expose `actions.createTemplate`. Render `InspectionTemplatesCard.svelte` + centered `CreateInspectionTemplateDialog.svelte` (`Dialog`, `sm:max-w-md`).

- [ ] **Step 2: Build `/inspections/incidents` route & components (`IncidentsTableCard.svelte`, `CreateIncidentDialog.svelte`, `EditIncidentDialog.svelte`)**

- Move entity-filtered `incidents` loading and `{ createIncident, updateIncident }` actions into `inspections/incidents/+page.server.ts`.
- Convert both Incident creation and Incident status/cost/observation editing from `Sheet` to centered `Dialog` modals (`CreateIncidentDialog.svelte`, `EditIncidentDialog.svelte`).
- Support `?incidentId=<id>` deep-link from Cockpit to auto-open `EditIncidentDialog` for the matching incident.

- [ ] **Step 3: Refactor `/inspections` route & components (`InspectionsListCard.svelte`, `CreateInspectionDialog.svelte`, `InspectionGridSheet.svelte`)**

- Trim `inspections/+page.server.ts` to Move-In / Move-Out inspections (`load` active leases, tenants, templates, inspections; `actions`: `{ createInspection, saveInspectionItems, generatePdf }`).
- Extract inspection creation into centered `CreateInspectionDialog.svelte` (`sm:max-w-lg`, calling `onCreated(inspection)` so the page immediately opens `InspectionGridSheet` on the new inspection). If `templates.length === 0`, show a clear link to `resolve('/(app)/inspections/templates')`.
- Keep `InspectionGridSheet.svelte` as `Sheet` (detail workspace #3) encapsulating room-by-room condition grading (`CONDITION_STATE_OPTIONS`), `saveInspectionItems`, and `generatePdf` report preview/download.

- [ ] **Step 4: Format, lint, test, and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/routes/\(app\)/inspections
git commit -m "feat(web): split inspections, incidents, and EDL templates into nested SOLID routes with centered modals"
```

---

### Task 7: Accounting & Tax Nested Routes (`/accounting`, `/accounting/journal`, `/accounting/tax`)

**Files:**
- Modify: `web/src/lib/modules/accounting/statement-state.ts`
- Modify: `web/src/lib/modules/accounting/statement-state.test.ts`
- Create: `web/src/lib/components/accounting/AccountingContextToolbar.svelte`
- Create: `web/src/routes/(app)/accounting/(components)/FinancialStatementsCard.svelte`
- Modify: `web/src/routes/(app)/accounting/+page.server.ts`
- Modify: `web/src/routes/(app)/accounting/+page.svelte`
- Create: `web/src/routes/(app)/accounting/journal/+page.server.ts`
- Create: `web/src/routes/(app)/accounting/journal/+page.svelte`
- Create: `web/src/routes/(app)/accounting/journal/(components)/TrialBalanceTableCard.svelte`
- Create: `web/src/routes/(app)/accounting/journal/(components)/JournalEntriesTableCard.svelte`
- Create: `web/src/routes/(app)/accounting/journal/(components)/UpsertBalanceLineDialog.svelte`
- Create: `web/src/routes/(app)/accounting/journal/(components)/CreateJournalEntryDialog.svelte`
- Create: `web/src/routes/(app)/accounting/tax/+page.server.ts`
- Create: `web/src/routes/(app)/accounting/tax/+page.svelte`
- Create: `web/src/routes/(app)/accounting/tax/(components)/AnnualAccountsPdfCard.svelte`
- Create: `web/src/routes/(app)/accounting/tax/(components)/FecExportCard.svelte`

**Interfaces:**
- Produces:
  - `evaluateJournalBalance(debitEuroInput: string, creditEuroInput: string): { debitCents: number; creditCents: number; balanced: boolean }` and `resolveAccountingContext(params: { entities: LegalEntityDTO[]; selectedEntityId: string; url: URL }): { activeEntityId: string; fiscalYear: number }` in `#lib/modules/accounting/statement-state`
  - `AccountingContextToolbar.svelte` in `#lib/components/accounting/` (reused across `/accounting`, `/accounting/journal`, `/accounting/tax`): `{ entities: LegalEntityDTO[]; activeEntityId: string; fiscalYear: number }`

- [ ] **Step 1: Write failing unit tests in `web/src/lib/modules/accounting/statement-state.test.ts`**

Assert:
- `evaluateJournalBalance('1 200,00', '1200,00')` → `{ debitCents: 120000, creditCents: 120000, balanced: true }`.
- `evaluateJournalBalance('0,00', '0,00')` → `balanced: false`.
- `evaluateJournalBalance('120,', '120,00')` and `evaluateJournalBalance('abc', '100')` never throw and return `balanced: false`.
- `resolveAccountingContext` falls back to `entities[0].id` when `selectedEntityId === 'ALL'` and parses `?year=2025` within `[2000, 2100]`.

- [ ] **Step 2: Run Vitest to confirm failure, then implement helpers in `statement-state.ts`**

Run: `npm --prefix web test` (confirm FAIL), implement `evaluateJournalBalance` & `resolveAccountingContext` in `statement-state.ts`, run `npm --prefix web run format`, and re-run `npm --prefix web test` (confirm PASS).

- [ ] **Step 3: Create shared `AccountingContextToolbar.svelte` and refactor `/accounting` (Financial Statements)**

- `AccountingContextToolbar.svelte`: Entity `Select` (`workspaceStore.setEntityId`) + GET `?year=` `Input` (`type="number"`, `min="2000"`, `max="2100"`).
- `accounting/+page.server.ts`: load only `previewStatements(activeEntityId, fiscalYear, disabledLineIds)` and expose `actions.recompute`.
- `accounting/(components)/FinancialStatementsCard.svelte` + slim `accounting/+page.svelte` (interactive Balance Sheet & Income Statement checkboxes triggering SSR `?/recompute`, summary KPIs, and quick link to `/accounting/tax?year=${data.fiscalYear}`).

- [ ] **Step 4: Build `/accounting/journal` route & centered modals (`UpsertBalanceLineDialog.svelte`, `CreateJournalEntryDialog.svelte`)**

- `accounting/journal/+page.server.ts`: load `[statements, journalEntries]` (`previewStatements` + `listJournalEntries`) and expose `{ upsertBalanceLine, recordJournalEntry }`.
- Extract `TrialBalanceTableCard.svelte`, `JournalEntriesTableCard.svelte`, centered `UpsertBalanceLineDialog.svelte` (`Dialog`, `sm:max-w-md`), and centered `CreateJournalEntryDialog.svelte` (`Dialog`, `sm:max-w-lg max-h-[90vh] overflow-y-auto`, using `evaluateJournalBalance`).

- [ ] **Step 5: Build `/accounting/tax` route & components (`AnnualAccountsPdfCard.svelte`, `FecExportCard.svelte`)**

- `accounting/tax/+page.server.ts`: load `previewStatements` summary and expose `actions.generatePdf`.
- Extract `AnnualAccountsPdfCard.svelte` (generates CERFA 2072/2033 / annual report PDF via `?/generatePdf` and displays download CTA) and `FecExportCard.svelte` (links to existing `/accounting/fec?entityId=...&fiscalYear=...&siren=...` normative 18-column DGFiP TSV stream; keep `web/src/routes/(app)/accounting/fec/+server.ts` untouched).

- [ ] **Step 6: Format, lint, test, and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/src/lib/modules/accounting web/src/lib/components/accounting web/src/routes/\(app\)/accounting
git commit -m "feat(web): split accounting statements, PCG journal, and tax/FEC exports into nested SOLID routes"
```

---

### Task 8: `Sidebar-07` Navigation (`AppSidebar`, `EntitySwitcher`, `SidebarUserMenu` & `sidebar-nav` Helpers)

**Files:**
- Create: `web/src/lib/components/layout/sidebar-nav.ts`
- Create: `web/src/lib/components/layout/sidebar-nav.test.ts`
- Create: `web/src/lib/components/layout/EntitySwitcher.svelte`
- Create: `web/src/lib/components/layout/SidebarUserMenu.svelte`
- Delete: `web/src/lib/components/layout/EntityWorkspaceSwitcher.svelte`
- Modify: `web/src/lib/components/layout/AppSidebar.svelte`
- Modify: `web/src/routes/(app)/+layout.svelte`

**Interfaces:**
- Consumes: Task 1 `formatEntityTypeLabel`; Task 2 `useEntityDialogContext`, `Collapsible`, `Avatar`; Tasks 4–7 all nested SvelteKit `RouteId` paths (`/(app)/leases/tenants`, `/(app)/leases/templates`, `/(app)/rents/cash-journal`, `/(app)/inspections/incidents`, `/(app)/inspections/templates`, `/(app)/accounting/journal`, `/(app)/accounting/tax`).
- Produces:
  - `isExactRouteActive(targetPath: string, currentPathname: string): boolean`, `isGroupPathActive(basePath: string, currentPathname: string): boolean`, and `formatUserInitials(fullName: string | null | undefined): string` in `#lib/components/layout/sidebar-nav`
  - `EntitySwitcher.svelte` props: `{ entities: LegalEntityDTO[] }`
  - `SidebarUserMenu.svelte` props: `{ user: UserDTO | null }`
  - `AppSidebar.svelte` props: `{ user?: UserDTO | null; entities?: LegalEntityDTO[] }`

- [ ] **Step 1: Write failing tests in `web/src/lib/components/layout/sidebar-nav.test.ts`**

Assert:
- `isGroupPathActive('/leases', '/leases')` is `true`, `isGroupPathActive('/leases', '/leases/tenants')` is `true`, `isGroupPathActive('/leases', '/leases-other')` is `false`.
- `isExactRouteActive('/leases', '/leases/tenants')` is `false` while `isExactRouteActive('/leases/tenants', '/leases/tenants/')` is `true` and `isExactRouteActive('/', '/')` is `true` (preventing parent and sub-item double-highlighting).
- `formatUserInitials('Yann Amsellem')` → `'YA'`, `formatUserInitials('Admin')` → `'AD'`, `formatUserInitials('')` → `'GI'`.

Run `npm --prefix web test` (confirm FAIL), implement `sidebar-nav.ts`, run `npm --prefix web run format`, and confirm Vitest PASS.

- [ ] **Step 2: Build `EntitySwitcher.svelte` and `SidebarUserMenu.svelte` (and delete `EntityWorkspaceSwitcher.svelte`)**

- `EntitySwitcher.svelte`: `Sidebar.Menu` > `Sidebar.MenuItem` > `DropdownMenu.Root` with `Sidebar.MenuButton size="lg" class="gap-3 text-sm"`, left `size-8 rounded-lg bg-primary text-primary-foreground` tile with `IconBuildingSkyscraper` (`class="size-5"`), entity name + `formatEntityTypeLabel`, `IconSelector` chevrons; dropdown lists entities (`⌘1`, `⌘2`, …), `Toutes les entités (${entities.length})` when `entities.length > 1`, separator, and `+ Créer une entité juridique…` calling `useEntityDialogContext().openCreateDialog()`.
- `SidebarUserMenu.svelte`: `NavUser` `DropdownMenu` with `Avatar.Root class="size-8 rounded-lg"` + `Avatar.Fallback` (`formatUserInitials(user?.full_name)`), user name + email, dropdown role badge, link to `resolve('/(app)/admin')` (*Administration & Accès MCP*, `IconShieldLock`), separator, and POST logout form (`resolve('/(auth)/logout')`).

- [ ] **Step 3: Rewrite `AppSidebar.svelte` in `sidebar-07` style (`collapsible="icon"`) with consumer-side Tailwind classes**

- `<Sidebar.Root collapsible="icon">`, consumer-side Tailwind classes `class="h-9 gap-3 text-sm [&_svg]:size-5 group-data-[collapsible=icon]:size-9!"` on `Sidebar.MenuButton`, `class="h-8 text-sm"` on `Sidebar.MenuSubButton`, and `class="text-xs font-medium"` on `Sidebar.GroupLabel` (zero edits to `web/src/lib/components/ui/**`).
- Use `resolve(...)` for every route link (`/(app)`, `/(app)/properties`, `/(app)/leases`, `/(app)/leases/tenants`, `/(app)/leases/templates`, `/(app)/rents`, `/(app)/rents/cash-journal`, `/(app)/inspections`, `/(app)/inspections/incidents`, `/(app)/inspections/templates`, `/(app)/entities`, `/(app)/accounting`, `/(app)/accounting/journal`, `/(app)/accounting/tax`).
- Sections:
  1. Standalone: `Vue d'ensemble` (`/(app)`, `IconLayoutDashboard`)
  2. `Biens & Exploitation`: `Biens & Lots` (`IconHome`), `Baux & Locataires` (`IconFileText`, `Collapsible`), `Loyers & Relances` (`IconReceiptEuro`, `Collapsible`), `États des lieux & Incidents` (`IconClipboardCheck`, `Collapsible`)
  3. `Société & Fiscalité`: `Entités & Associés` (`IconBuildingBank`), `Comptabilité & Fiscalité` (`IconCalculator`, `Collapsible`)
- Update `(app)/+layout.svelte` `<AppSidebar user={data.user} entities={data.entities} />` (removing obsolete `openIncidentsCount` prop).

- [ ] **Step 4: Format, lint, test, and typecheck**

Run: `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`
Expected: PASS (0 errors, 0 ESLint warnings).

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/components/layout web/src/routes/\(app\)/+layout.svelte
git commit -m "feat(web): upgrade AppSidebar to sidebar-07 collapsible sections, EntitySwitcher, and SidebarUserMenu"
```

---

### Task 9: Final Architecture Audit (`ui/` Immutability, Overlay Policy, Line Counts & Production Build)

**Files:**
- Verify across `web/` and `api/`

- [ ] **Step 1: Verify `web/src/lib/components/ui/**` immutability and overlay policy**

Run:
- `git diff 06485d9 -- web/src/lib/components/ui` → verify ONLY the newly added `avatar/` and `collapsible/` directories exist and ZERO pre-existing `ui/` files were modified.
- `grep -rn "Sheet\." web/src/routes web/src/lib/components/layout web/src/lib/components/portfolio` → verify `Sheet.*` appears ONLY in the 3 approved detail workspaces (`LeaseDocumentsSheet.svelte`, `RentActionSheet.svelte`, `InspectionGridSheet.svelte`).
- `grep -rn "searchParams.get('tab')\|searchParams.get('view')" web/src` → verify 0 matches.
- `wc -l web/src/routes/\(app\)/**/+page.svelte` → verify every `+page.svelte` is a lean orchestrator (< 200 lines) with zero form-field `$state` bloat.

- [ ] **Step 2: Run full frontend & backend verification gates**

Run:
1. `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check && npm --prefix web run build`
2. `make -C api test && make -C api lint && make -C api build`
Expected: All commands exit `0` with 0 errors and 0 warnings.

- [ ] **Step 3: Commit any final formatting/cleanup if needed**

If working tree has any remaining changes, commit cleanly; otherwise confirm `git status` is clean.
