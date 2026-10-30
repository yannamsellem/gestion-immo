# Frontend UX, Sidebar-07 Navigation, Persistent Onboarding & SOLID Component Architecture — Design Spec

- **Date:** 2026-10-30
- **Scope:** SvelteKit 2 + Svelte 5 frontend (`web/`)
- **Status:** Updated after user review

---

## 1. Intent & Why

The `web/` back-office currently suffers from four UX and architectural friction points:

1. **Cramped, flat sidebar navigation:** Built on the `mira` preset (`12px` / `text-xs` menu items and `16px` icons in a single flat list of 8 links), the sidebar lacks visual hierarchy, readable typography, and domain grouping.
2. **No mandatory starting point ("Point de départ"):** In real estate & corporate wealth management, **nothing** exists without a Legal Entity (`LegalEntity` — SCI IS/IR, SARL, SAS, Nom propre, Indivision). Currently, a newly authenticated user with zero legal entities lands on an empty dashboard and can navigate across pages that cannot function without an entity.
3. **Overuse of side panels (`Sheet`) for resource creation:** Creating entities, shareholders, properties, lots, leases, tenants, incidents, and accounting lines currently opens right-hand `Sheet` drawers instead of focused, centered `Dialog` modals.
4. **Monolithic `+page.svelte` & `+page.server.ts` files violating SRP:** Route pages (`leases/+page.svelte` at 1,076 lines, `inspections/+page.svelte` at 1,078 lines, `accounting/+page.svelte` at 795 lines, `rents/+page.svelte` at 698 lines, `entities/+page.svelte` at 584 lines, `properties/+page.svelte` at 577 lines) bundle multiple distinct sub-domains behind client-side tabs and mix page layout, KPI computation, multiple tables, and 15–30 `$state` variables for unrelated creation/edit forms.

This design transforms the frontend around a `shadcn-svelte` `sidebar-07` navigation backed by dedicated SvelteKit nested routes ordered by natural domain dependencies, a persistent non-cancelable first-run Legal Entity creation modal, centered `Dialog` modals for all creation/edit forms, and small **Clean Code / SOLID** Svelte 5 components colocated cleanly — without ever hand-editing generated `shadcn-svelte` primitives in `web/src/lib/components/ui/`.

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
1. **Refined `sidebar-07` Navigation (`collapsible="icon"`) via Consumer-Side Tailwind Classes:**
   - Larger, breathable typography (`text-sm` / `14px` menu items, `h-9` button height) and `@tabler/icons-svelte` icons (`size-5` / `20px`) with `gap-3` (`12px`) spacing between icon and label — applied via Tailwind `class` overrides in `AppSidebar.svelte` / `EntitySwitcher.svelte` / `SidebarUserMenu.svelte` (leveraging `cn()` / `tailwind-merge`), keeping `web/src/lib/components/ui/**` 100% untouched.
   - Top **Entity Switcher** (`DropdownMenu`, `TeamSwitcher` pattern from `sidebar-07`) displaying the active Legal Entity (or consolidated view `Toutes les entités` when `entities.length > 1`) plus a **`+ Créer une entité juridique…`** action opening the shared `CreateEntityDialog` modal from any screen.
   - Logical dependency sections (**Vue d'ensemble**, **Biens & Exploitation**, **Société & Fiscalité**) with `Collapsible` sub-menus (`Sidebar.MenuSub`) pointing to **dedicated SvelteKit nested routes** (`/leases`, `/leases/tenants`, `/leases/templates`, etc.), eliminating in-page tab bars.
   - Bottom **User Menu** (`NavUser` `DropdownMenu` with `Avatar` + name + email) housing **Administration & Accès** (`/admin`) and **Se déconnecter**.
2. **Persistent Legal Entity Onboarding Gate:**
   - Whenever `data.entities.length === 0 && !data.entitiesLoadError`, automatically open the shared `CreateEntityDialog` in **persistent mode** (`persistent={true}`): no close `×` button, no *Annuler* button, `escapeKeydownBehavior="ignore"`, and `interactOutsideBehavior="ignore"`.
   - Upon creation, automatically activate the newly created entity (`gi_entity_id` cookie + `workspaceStore.setEntityId`) and unlock the workspace.
3. **Overlay Policy (Option B — Centered Modals for Forms, Side Sheets for Multi-Action Detail Workspaces):**
   - **Centered `Dialog` modals** for **all** resource creation and edit forms (15 forms across the app).
   - **Right-hand `Sheet` side panels** preserved **exclusively** for the 3 multi-action detail workspaces tied to an existing table row:
     1. `LeaseDocumentsSheet` (generate `.docx`/PDF from template, upload file to R2, list/download archived files for a selected lease).
     2. `RentActionSheet` (record payment, generate receipt/formal notice, assemble GLI ZIP dossier for a selected rent call/lease).
     3. `InspectionGridSheet` (qualify room-by-room elements, save grid, generate contradictory Move-In/Move-Out PDF report).
4. **Strict Component Placement Convention:**
   - Components reused in **2+ places** live under `web/src/lib/components/` (organized by domain/layer: `ui/`, `layout/`, `portfolio/`).
   - Components used by a **single route or layout** live in a colocated `(components)/` directory alongside that exact `+page.svelte` or `+layout.svelte` (e.g., `web/src/routes/(app)/leases/(components)/CreateLeaseDialog.svelte`, `web/src/routes/(app)/leases/tenants/(components)/CreateTenantDialog.svelte`).
5. **SOLID & Clean Code Component & Route Design:**
   - Treat every SvelteKit route (`+page.server.ts` + `+page.svelte`) and every Svelte 5 component as a focused function/object with a single responsibility, narrow typed `$props()`, encapsulated local `$state()`, and explicit callback props.

### Non-Goals
- Modifying Go backend domain logic, SQL schemas, or Huma OpenAPI endpoints.
- Introducing client-side SPA state stores or replacing SvelteKit SSR + Form Actions (`use:enhance`).

### Anti-Goals
- **Never Edit `web/src/lib/components/ui/**` Directly:** Never hand-modify generated `shadcn-svelte` library components, because future `npx shadcn-svelte@latest add` / `update` runs can overwrite them and silently break custom UI styling. Always customize sizing, typography, spacing, and layout via `class="..."` props or wrapper components outside `ui/`.
- **Never Use Query Parameters for Sub-View Routing:** Never use `?view=...` or `?tab=...` to switch between distinct sub-views. Use real RESTful SvelteKit paths (`/leases/tenants`, `/accounting/journal`, etc.). Reserve URL query parameters strictly for shared client/server filter, search, pagination, or pre-fill state (`?year=2026`, `?q=...`, `?page=1`, `?lotId=...`, `?incidentId=...`).
- **Zero Franglais (`AGENTS.md` Rule 1):** All filenames, route directory names, component names, prop names, variables, and types MUST be 100% English. French is strictly confined to rendered UI labels and `#lib/modules/*/labels.ts` display dictionaries.
- **Zero Floating-Point Money (`AGENTS.md` Rule 2):** Never transport or compute money in floats; parse user euro inputs into integer cents on the server (`parseEuroToCents`) and format cents/basis points via `#lib/core/formatters/money`.
- **Zero God-Pages:** No `+page.svelte` may hold raw form-field `$state` variables belonging to a modal or drawer.
- **No False Onboarding Lock on Backend Error:** A failed `listEntities()` API call (`500` / network outage) must render an explicit error banner with retry/logout — never lock the user inside the persistent `CreateEntityDialog`.

---

## 3. Hard Constraints & Quality Gates

- **Immutable `ui/` Directory:** Only `npx shadcn-svelte@latest add collapsible avatar` adds files into `web/src/lib/components/ui/`. Existing `ui/` files must remain untouched (`git diff web/src/lib/components/ui` shows only the newly added official `collapsible/` and `avatar/` folders).
- **Icon Library:** Strictly `@tabler/icons-svelte` (per `web/components.json` `"iconLibrary": "tabler"`).
- **Mandatory Immediate Formatting (`AGENTS.md` Rule 5):** Run `npm --prefix web run format` immediately after creating or modifying `.ts`, `.svelte`, or `.css` files in `web/`.
- **Verification Suite:** `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check && npm --prefix web run build` (plus `make -C api test && make -C api lint && make -C api build` if any backend file is touched).

---

## 4. SOLID Principles Applied to SvelteKit Routes & Svelte 5 Components

Every new or refactored route, `.svelte` component, and `.ts` module must adhere to SOLID:

1. **S — Single Responsibility Principle (SRP):**
   - Each nested route (`+page.server.ts` + `+page.svelte`) serves **one** cohesive sub-domain: e.g., `/leases/tenants` only loads `listTenants()` and exposes `actions.createTenant`; `/leases/templates` only loads `listTemplates()` and exposes `actions.registerTemplate`.
   - `+page.svelte` is strictly a **route orchestrator**: it reads `data`, renders `PageHeader` + summary cards + the route's main table/card, and binds `open` / selected item references to dialogs/sheets.
   - Each `*Dialog.svelte` owns **only** its own form fields (`$state`), submission loading flag (`isSubmitting`), validation feedback, and form reset logic on success.
   - Each `*TableCard.svelte` / `*Card.svelte` owns **only** rendering a typed list/entity and emitting user-intent callbacks (`onSelectEntity`, `onAddLot`, `onEditIncident`, `onOpenDocuments`, etc.).
   - Domain enum-to-French-label mappings live in pure TypeScript modules (`#lib/modules/<domain>/labels.ts`), never duplicated inline inside `.svelte` files.
2. **O — Open/Closed Principle (OCP):**
   - Shared components such as `CreateEntityDialog.svelte` extend behavior via props (`persistent?: boolean`, `onCreated?: (entity: LegalEntityDTO) => void`) rather than inspecting `$page.url.pathname` or hardcoding caller-specific branches.
   - `shadcn-svelte` UI primitives in `web/src/lib/components/ui/` are extended via `class` composition (`cn()`) and wrapper components rather than modifying their source files.
3. **L — Liskov Substitution Principle (LSP):**
   - All modal form components follow a uniform behavioral contract: `open = $bindable(false)`, `use:enhance` progressive enhancement that inspects `result` (`result.type === 'success' | 'failure' | 'error'`), displays `toast.success` / `toast.error`, refreshes server data (`invalidateAll()` or `update()`), resets its internal fields, and closes itself cleanly when `result.type === 'success'`.
4. **I — Interface Segregation Principle (ISP):**
   - Never pass `data: PageData` or `form: ActionData` mega-objects into child components. Pass only the minimal typed props required (e.g., `entities: LegalEntityDTO[]`, `properties: PropertyDTO[]`, `selectedIncident: IncidentDTO | null`).
5. **D — Dependency Inversion Principle (DIP):**
   - Leaf UI components depend on domain DTO/enum types (`#lib/modules/*/types`) and callback props rather than reaching into route-level `$types` or mutating sibling state directly.

---

## 5. Sidebar-07 Architecture & Nested Route Tree

### 5.1 Consumer-Side Tailwind Typography & Icon Sizing (Zero Edits to `ui/`)
Inside `AppSidebar.svelte`, `EntitySwitcher.svelte`, and `SidebarUserMenu.svelte`, apply Tailwind override classes directly on `Sidebar.*` components (`tailwind-merge` cleanly overrides `mira`'s `h-8 text-xs gap-2 [&_svg]:size-4` defaults):
- Primary menu buttons (`<Sidebar.MenuButton class="h-9 gap-3 text-sm [&_svg]:size-5 group-data-[collapsible=icon]:size-9!">`): `14px` text, `20px` Tabler icons, `12px` icon-to-text gap.
- Header/Footer switcher buttons (`<Sidebar.MenuButton size="lg" class="gap-3 text-sm">`): spacious `48px` (`h-12`) button height.
- Sub-menu buttons (`<Sidebar.MenuSubButton class="h-8 text-sm">`): `14px` readable sub-navigation items.
- Group labels (`<Sidebar.GroupLabel class="text-xs font-medium">`): `12px` section headers.
- Root sidebar (`<Sidebar.Root collapsible="icon">`): collapses on desktop to a clean `sidebar-07` icon rail with tooltips (`tooltipContent={item.label}`), while mobile continues using the built-in offcanvas drawer.

### 5.2 Navigation Sections, Collapsible Sub-Menus & Dedicated Routes

Ordered strictly from foundational resources to operational & tax outputs:

1. **Header — `EntitySwitcher.svelte` (`DropdownMenu`):**
   - Left tile (`size-8 rounded-lg bg-primary text-primary-foreground`) with `IconBuildingSkyscraper` (`size-5`).
   - Primary line: active entity `name` (or `Toutes les entités` when `selectedEntityId === 'ALL'`).
   - Secondary line: French label of `entity_type` (or `${entities.length} structures actives`).
   - Dropdown content:
     - List of legal entities (`⌘1`, `⌘2`, … visual shortcut hint or check indicator; clicking calls `workspaceStore.setEntityId(entity.id)`).
     - When `entities.length > 1`, an option `Toutes les entités (${entities.length})` (`workspaceStore.setEntityId('ALL')`).
     - `DropdownMenu.Separator` + **`+ Créer une entité juridique…`** item (`IconPlus`) that invokes `openCreateEntityDialog()`.

2. **Top Standalone Item:**
   - **Vue d'ensemble** (`/` — `IconLayoutDashboard`)

3. **Section 1 — `Biens & Exploitation` (`Sidebar.Group`):**
   - **Biens & Lots** (`/properties` — `IconHome`) — direct route.
   - **Baux & Locataires** (`IconFileText`, `Collapsible` open by default when pathname starts with `/leases`):
     - `Baux actifs` → `/leases`
     - `Locataires` → `/leases/tenants`
     - `Modèles de documents` → `/leases/templates`
   - **Loyers & Relances** (`IconReceiptEuro`, `Collapsible` open by default when pathname starts with `/rents`):
     - `Appels & Encaissements` → `/rents`
     - `Livre de caisse (530)` → `/rents/cash-journal`
   - **États des lieux & Incidents** (`IconClipboardCheck`, `Collapsible` open by default when pathname starts with `/inspections`):
     - `États des lieux` → `/inspections`
     - `Incidents & Sinistres` → `/inspections/incidents`
     - `Modèles d'EDL` → `/inspections/templates`

4. **Section 2 — `Société & Fiscalité` (`Sidebar.Group`):**
   - **Entités & Associés** (`/entities` — `IconBuildingBank`) — direct route.
   - **Comptabilité & Fiscalité** (`IconCalculator`, `Collapsible` open by default when pathname starts with `/accounting`):
     - `Bilan & Compte de résultat` → `/accounting`
     - `Journal & Balance PCG` → `/accounting/journal`
     - `Liasse fiscale & Export FEC` → `/accounting/tax` *(existing TSV download endpoint `/accounting/fec/+server.ts` remains at `/accounting/fec`)*

5. **Footer — `SidebarUserMenu.svelte` (`NavUser` `DropdownMenu`):**
   - Displays `Avatar` (initials derived from `user.full_name`), `user.full_name`, and `user.email` with `IconSelector` chevrons.
   - Dropdown content:
     - User role badge (`user.role`).
     - Link item: **Administration & Accès MCP** (`/admin`, `IconShieldLock`).
     - `DropdownMenu.Separator`
     - POST form to `/(auth)/logout`: **Se déconnecter** (`IconLogout`).

### 5.3 Deep-Link Query Parameters (Strictly Shared Filter/Pre-Fill State)
Cross-page links use query parameters **only** for shared filter or pre-fill state:
- From Cockpit (`/`) open incident row → `/inspections/incidents?incidentId=<id>` (pre-selects & opens `EditIncidentDialog` for that incident).
- From Properties (`/properties`) vacant lot row → `/leases?entityId=<id>&lotId=<id>` (pre-selects the vacant lot and opens `CreateLeaseDialog`).
- From Properties (`/properties`) occupied lot row → `/rents?entityId=<id>&leaseId=<id>` (highlights/opens that lease's rent call).
- Fiscal year filter on `/accounting` and `/accounting/tax` → `?year=2026`.

---

## 6. Persistent First-Run Legal Entity Onboarding Gate

### 6.1 Server Load Safety (`web/src/routes/(app)/+layout.server.ts`)
- Keep layout SSR single-responsibility and fast: load only `locals.api.portfolio.listEntities()` (no global incident/payment fan-out on every route transition).
- Distinguish an empty portfolio from an API failure:
  - Return `entitiesLoadError: string | null` (`null` on success, error message on non-401 failure).
- Keep `'ALL'` valid when multiple entities exist, and whenever the user creates an entity via `createEntity`, set `WORKSPACE_ENTITY_COOKIE` to `created.id` on the server so the newly created entity becomes immediately active.

### 6.2 Layout Gate (`web/src/routes/(app)/+layout.svelte`)
- Derive `requiresEntityOnboarding = $derived(data.entities.length === 0 && !data.entitiesLoadError)`.
- When `data.entitiesLoadError` is non-null and `data.entities.length === 0`:
  - Render an `Alert` error state inside `<main>` with a retry button (`invalidateAll()`) and logout button, **without** triggering the persistent creation modal.
- When `requiresEntityOnboarding` is `true`:
  - Do **not** render `{@render children()}` underneath; render a clean welcome backdrop with a discreet top-right `ThemeToggle` + `Se déconnecter` button.
  - Mount `<CreateEntityDialog open={true} persistent={true} />`:
    - `Dialog.Root` ignores close attempts (`onOpenChange` keeps `open = true` while `persistent` is `true`).
    - `Dialog.Content` sets `showCloseButton={false}`, `escapeKeydownBehavior="ignore"`, and `interactOutsideBehavior="ignore"`.
    - Title & description adapt to first-run onboarding (*« Bienvenue — Créez votre première entité juridique »*) with no *Annuler* button and a full-width primary submit button (*« Créer l'entité juridique et démarrer »*).
- When `requiresEntityOnboarding` is `false`:
  - Render `Sidebar.Provider` + `AppSidebar` + header + `{@render children()}`.
  - Mount `<CreateEntityDialog bind:open={createEntityDialogOpen} persistent={false} />` once at the `(app)/+layout.svelte` level and expose `openCreateEntityDialog()` via `#lib/core/workspace/entity-dialog-context.ts`.

### 6.3 Cross-Route Form Submission in `CreateEntityDialog.svelte`
- Posts to `action="{resolve('/(app)/entities')}?/createEntity"` using `use:enhance`.
- In `entities/+page.server.ts` `actions.createEntity`:
  - On success, call `setWorkspaceCookies(cookies, { selectedEntityId: created.id }, url.protocol === 'https:')` before returning `{ createdEntity: created, successMessage: ... }`.
- In `CreateEntityDialog.svelte`'s `use:enhance` callback:
  - Inspect `result` directly:
    - `result.type === 'success'`: show `toast.success`, update `workspaceStore.setEntityId(created.id)` (which triggers `invalidateAll()`), reset local fields, invoke `onCreated?.(created)`, and set `open = false` if `!persistent` (when `persistent` was `true`, `invalidateAll()` updates `data.entities` to length 1, flipping `requiresEntityOnboarding` to `false` automatically).
    - `result.type === 'failure'` / `'error'`: keep modal open and display inline error + `toast.error`.

---

## 7. Component & Route Inventory

### 7.1 Shared Domain Labels (`web/src/lib/modules/*/labels.ts`) — DRY & Bug Fixes
Centralize French UI labels for backend OpenAPI enums (100% English exports, French display strings):
- `web/src/lib/modules/portfolio/labels.ts`: `ENTITY_TYPE_OPTIONS`, `formatEntityTypeLabel`, `PROPERTY_TYPE_OPTIONS`, `ACCOUNTING_CATEGORY_OPTIONS`, `LOT_TYPE_OPTIONS`, `formatLotTypeLabel` *(fixes stale `SARL_FAMILLE`/`LMNP`/`NOM_PROPRE` labels in `(app)/+page.svelte`)*.
- `web/src/lib/modules/lease/labels.ts`: `LEASE_TYPE_OPTIONS`, `formatLeaseTypeLabel`, `INDEXATION_TYPE_OPTIONS`.
- `web/src/lib/modules/document/labels.ts`: `TEMPLATE_TYPE_OPTIONS`, `formatTemplateTypeLabel` *(fixes stale `'GLI_NOTICE'`/`'REMINDER'`/`'RECEIPT'` lookups in `RentActionSheet.svelte` to `'GLI_DECLARATION'`/`'FORMAL_NOTICE'`/`'RENT_RECEIPT'`)*.
- `web/src/lib/modules/payment/labels.ts`: `PAYMENT_METHOD_OPTIONS`, `formatPaymentMethodLabel`.
- `web/src/lib/modules/inspection/labels.ts`: `INCIDENT_STATUS_OPTIONS`, `formatIncidentStatusLabel`, `formatIncidentStatusVariant`, `INSPECTION_TYPE_OPTIONS`, `formatInspectionTypeLabel`, `CONDITION_STATE_OPTIONS`.

### 7.2 Shared Components (`web/src/lib/components/`)
Used across 2+ routes or global app chrome:
- `web/src/lib/components/ui/collapsible/*` & `web/src/lib/components/ui/avatar/*` — official `shadcn-svelte` primitives added via CLI and never hand-edited.
- `web/src/lib/components/layout/AppSidebar.svelte` — `sidebar-07` shell composing `EntitySwitcher`, collapsible section groups, and `SidebarUserMenu`.
- `web/src/lib/components/layout/EntitySwitcher.svelte` — replaces `EntityWorkspaceSwitcher.svelte` (deleted). `TeamSwitcher`-style `DropdownMenu` with `+ Créer une entité juridique…`.
- `web/src/lib/components/layout/SidebarUserMenu.svelte` — `NavUser`-style `DropdownMenu` with `Avatar`, `/admin` link, and logout action.
- `web/src/lib/components/layout/PageHeader.svelte` & `ThemeToggle.svelte` — existing shared layout helpers.
- `web/src/lib/components/portfolio/CreateEntityDialog.svelte` — shared centered `Dialog` modal used by onboarding, `EntitySwitcher`, and `/entities`.

*(Delete obsolete `web/src/lib/components/layout/EntityWorkspaceSwitcher.svelte` and `web/src/lib/components/rents/RentActionSheet.svelte` after relocating `RentActionSheet.svelte` to `rents/(components)/`.)*

### 7.3 Routes & Route-Colocated Components (`(components)/` per Route)

1. **Cockpit — `web/src/routes/(app)/`**
   - `+page.server.ts` & `+page.svelte`
   - `(components)/CockpitMetricsGrid.svelte`
   - `(components)/PortfolioCockpitGroupCard.svelte`

2. **Entities & Shareholders — `web/src/routes/(app)/entities/`**
   - `+page.server.ts` & `+page.svelte`
   - `(components)/EntitiesMetricsGrid.svelte`
   - `(components)/EntitiesTableCard.svelte`
   - `(components)/ShareholdersTableCard.svelte`
   - `(components)/UpsertShareholderDialog.svelte` — **centered `Dialog` modal**

3. **Properties & Lots — `web/src/routes/(app)/properties/`**
   - `+page.server.ts` & `+page.svelte`
   - `(components)/PropertyLotsCard.svelte`
   - `(components)/CreatePropertyDialog.svelte` — **centered `Dialog` modal**
   - `(components)/CreateLotDialog.svelte` — **centered `Dialog` modal**

4. **Leases Bounded Context — `web/src/routes/(app)/leases/`**
   - **Active Leases (`/leases`):**
     - `leases/+page.server.ts` & `leases/+page.svelte`
     - `leases/(components)/LeasesTableCard.svelte`
     - `leases/(components)/CreateLeaseDialog.svelte` — **centered `Dialog` modal** (`sm:max-w-lg`)
     - `leases/(components)/LeaseDocumentsSheet.svelte` — **preserved `Sheet` side drawer** (detail workspace #1)
   - **Tenants (`/leases/tenants`):**
     - `leases/tenants/+page.server.ts` & `leases/tenants/+page.svelte`
     - `leases/tenants/(components)/TenantsTableCard.svelte`
     - `leases/tenants/(components)/CreateTenantDialog.svelte` — **centered `Dialog` modal**
   - **Document Templates (`/leases/templates`):**
     - `leases/templates/+page.server.ts` & `leases/templates/+page.svelte`
     - `leases/templates/(components)/DocumentTemplatesCard.svelte`
     - `leases/templates/(components)/RegisterTemplateDialog.svelte` — **centered `Dialog` modal**

5. **Rents & Cash Ledger — `web/src/routes/(app)/rents/`**
   - **Rent Calls & Collections (`/rents`):**
     - `rents/+page.server.ts` & `rents/+page.svelte`
     - `rents/(components)/RentsMetricsGrid.svelte`
     - `rents/(components)/RentCallsTableCard.svelte`
     - `rents/(components)/ImportPaymentsCsvDialog.svelte` — **centered `Dialog` modal**
     - `rents/(components)/RentActionSheet.svelte` — **preserved `Sheet` side drawer** (detail workspace #2)
   - **PCG 530 Cash Journal (`/rents/cash-journal`):**
     - `rents/cash-journal/+page.server.ts` & `rents/cash-journal/+page.svelte`
     - `rents/cash-journal/(components)/CashJournalTableCard.svelte`
     - `rents/cash-journal/(components)/CashDepositDialog.svelte` — **centered `Dialog` modal**

6. **Inspections & Incidents — `web/src/routes/(app)/inspections/`**
   - **Move-In / Move-Out Inspections (`/inspections`):**
     - `inspections/+page.server.ts` & `inspections/+page.svelte`
     - `inspections/(components)/InspectionsListCard.svelte`
     - `inspections/(components)/CreateInspectionDialog.svelte` — **centered `Dialog` modal** (`sm:max-w-lg`)
     - `inspections/(components)/InspectionGridSheet.svelte` — **preserved `Sheet` side drawer** (detail workspace #3)
   - **Incidents & Claims (`/inspections/incidents`):**
     - `inspections/incidents/+page.server.ts` & `inspections/incidents/+page.svelte`
     - `inspections/incidents/(components)/IncidentsTableCard.svelte`
     - `inspections/incidents/(components)/CreateIncidentDialog.svelte` — **centered `Dialog` modal**
     - `inspections/incidents/(components)/EditIncidentDialog.svelte` — **centered `Dialog` modal**
   - **Inspection Templates (`/inspections/templates`):**
     - `inspections/templates/+page.server.ts` & `inspections/templates/+page.svelte`
     - `inspections/templates/(components)/InspectionTemplatesCard.svelte`
     - `inspections/templates/(components)/CreateInspectionTemplateDialog.svelte` — **centered `Dialog` modal**

7. **Accounting & Tax — `web/src/routes/(app)/accounting/`**
   - **Financial Statements (`/accounting`):**
     - `accounting/+page.server.ts` & `accounting/+page.svelte`
     - `accounting/(components)/AccountingFiscalYearToolbar.svelte` (or shared in `accounting/(components)/` / `#lib/components/accounting/` if reused across `/accounting` and `/accounting/tax`)
     - `accounting/(components)/FinancialStatementsCard.svelte`
   - **PCG Journal & Trial Balance (`/accounting/journal`):**
     - `accounting/journal/+page.server.ts` & `accounting/journal/+page.svelte`
     - `accounting/journal/(components)/TrialBalanceTableCard.svelte`
     - `accounting/journal/(components)/UpsertBalanceLineDialog.svelte` — **centered `Dialog` modal**
     - `accounting/journal/(components)/CreateJournalEntryDialog.svelte` — **centered `Dialog` modal** (`sm:max-w-lg`)
   - **Tax Package & FEC Export (`/accounting/tax`):**
     - `accounting/tax/+page.server.ts` & `accounting/tax/+page.svelte`
     - `accounting/tax/(components)/AnnualAccountsPdfCard.svelte`
     - `accounting/tax/(components)/FecExportCard.svelte`
     - `accounting/fec/+server.ts` — existing normative 18-column DGFiP FEC `.tsv` download endpoint (unchanged).

8. **Administration & MCP Access — `web/src/routes/(app)/admin/`**
   - `admin/+page.server.ts` & `admin/+page.svelte`
   - `admin/(components)/McpTokensCard.svelte`
   - `admin/(components)/PendingUsersGovernanceCard.svelte`

*(Note on component reuse inside `/accounting`: if `FiscalYearSelector` is used on both `/accounting` and `/accounting/tax`, place it in `web/src/lib/components/accounting/FiscalYearSelector.svelte` per the 2+ routes placement rule.)*

---

## 8. Approved Design Decisions

1. **Svelte Context for Global `CreateEntityDialog`:** `(app)/+layout.svelte` owns the single `<CreateEntityDialog>` instance and exposes a typed context helper (`#lib/core/workspace/entity-dialog-context.ts`: `setEntityDialogContext` / `useEntityDialogContext`).
2. **Dedicated Nested SvelteKit Routes:** Sub-menus use real path-based routes (`/leases`, `/leases/tenants`, `/leases/templates`; `/rents`, `/rents/cash-journal`; `/inspections`, `/inspections/incidents`, `/inspections/templates`; `/accounting`, `/accounting/journal`, `/accounting/tax`). Query parameters are reserved exclusively for shared filter/pre-fill state (`?year=`, `?lotId=`, `?leaseId=`, `?incidentId=`).
3. **Immutable `web/src/lib/components/ui/`:** Zero modifications to generated `shadcn-svelte` primitives; all `14px` text, `20px` Tabler icons, and `gap-3` spacing adjustments are applied via consumer-side Tailwind classes.
4. **Dialog Sizing & Scroll Safety:** Standard single-column creation modals use `sm:max-w-md`; denser 2-column modals use `sm:max-w-lg max-h-[90vh] overflow-y-auto`.
5. **Logout Escape Hatch During Persistent Onboarding:** A discreet top-bar *Se déconnecter* button remains accessible above the persistent onboarding backdrop.
6. **Uncluttered `sidebar-07` Rows Without Counter Badges:** Collapsible rows display only their disclosure chevron (`IconChevronRight`), and `(app)/+layout.server.ts` loads only `listEntities()`.
