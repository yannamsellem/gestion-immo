# Design Specification — `gestion-immo` Web Frontend (Svelte 5 & shadcn-svelte)

- **Date:** 2026-10-27
- **Scope:** `/web` (SvelteKit 3, Svelte 5 Runes, Tailwind CSS v4, `shadcn-svelte` preset `mira` / `zinc` / `tabler`)
- **Status:** Ready for User Approval (Builder Check Passed)

---

## 1. Intent & Why

The `gestion-immo` web application is the daily operational back-office for managing a multi-entity French real-estate portfolio (SCI à l'IS/IR, LMNP, détention en nom propre).

When opening the application, the user does not want passive analytics charts or vanity metrics. They want to **see at a glance the global status of their portfolio structured by Legal Entity (entreprise) and by Property/Lot (bien), so they immediately know what work is waiting for them** (generating monthly rent calls and provisions, recording incoming rent payments, sending overdue rent reminders / assembling GLI claim dossiers, and handling property incidents) and can jump straight to executing each task via a direct Call-to-Action (CTA).

---

## 2. Goals, Non-Goals & Anti-Goals

### Goals
1. **Action-Oriented Operational Cockpit (`/`):** Display every Legal Entity and its Properties/Lots with live monthly collection progress and inline CTAs on each property/lot row for pending tasks (`Générer appels`, `Saisir loyer`, `Relancer impayé`, `Incident ouvert`).
2. **Global Entity Workspace Switcher + Domain Sidebar:** Allow filtering the entire application by `"Toutes les entités"` or a single Legal Entity from the top of the sidebar, paired with clear domain navigation and a **Dark / Light mode switch**.
3. **Unified Table + Contextual Right `Sheet` Workflow:** Across operational screens (`Loyers & Relances`, `Incidents & États des lieux`, `Biens & Lots`, `Baux & Locataires`, `Entités & Associés`), clicking a Cockpit CTA or a table row opens a right-hand `shadcn-svelte` `Sheet` pre-populated with the targeted lease, property, or incident to complete the task immediately.
4. **Interactive Annual Accounts & Tax Package (`/accounting`):** Provide double-entry PCG journal entry management, interactive financial statements (`Bilan`, `Compte de résultat`, `SIG`) where individual lines can be toggled on/off with live recalculation, CERFA 2072/2033 PDF generation, and normative 18-column DGFiP FEC `.tsv` export.
5. **Complete Authentication & Admin Governance:** Support login, signup (`202 Accepted` pending approval state), automatic JWT refresh, `SUPER_ADMIN` user approval/rejection, and MCP bearer token generation.
6. **Exemplary Svelte 5, Clean Code & SOLID Architecture:** Use Svelte 5 runes (`$state`, `$derived`, `$effect`, `$props`, `$bindable`, snippets), `shadcn-svelte` CLI components (`mira` style, `zinc` base color, `@tabler/icons-svelte`), 100% English source code identifiers, French UI text, and integer-only monetary/percentage DTO transport.

### Non-Goals
- Modifying the Go backend (`api/`) or SQL migrations (the REST API at `/api/v1/*` is already complete and tested).
- Building a tenant-facing self-service portal (this is strictly a manager/owner back-office).

### Anti-Goals (Failure Conditions Even If It Technically Works)
- **Franglais in Code:** Any French identifier in TypeScript types, variables, props, functions, stores, route slugs, or file names violates `AGENTS.md`.
- **Floating-Point Money Manipulation:** Performing arithmetic in floating-point euros or storing formatted strings in DTOs instead of using integer `AmountCents` (`*_cents`) and `BasisPoints` (`*_basis_points`) formatted only at the UI boundary.
- **Wrong Import Alias:** Importing from `$lib/*` instead of `#lib/*` (`web/package.json` and `web/components.json` define subpath imports `#lib` and `#lib/*`).
- **God Components:** Mixing raw `fetch()` calls, business state orchestration, and UI presentation inside `.svelte` route files instead of separating API clients, reactive domain services/stores, and presentational components.
- **Svelte 4 Legacy Syntax:** Using `export let`, `$:`, `<slot>`, `on:click`, or `createEventDispatcher` (strict Svelte 5 runes mode is enforced in `web/vite.config.ts`).

---

## 3. Hard Constraints (`AGENTS.md` & Project Bootstrap)

1. **Language & Localization Boundary:**
   - **Code (100% English):** All files, directories, route slugs (`/login`, `/signup`, `/rents`, `/inspections`, `/properties`, `/leases`, `/entities`, `/accounting`, `/admin`), TypeScript interfaces, classes, functions, props, and tests must be in English.
   - **UI Display (100% French):** All labels, headings, button captions, status badges, error messages, and formatted dates/numbers shown to the user must be in natural French.
2. **Monetary & Percentage Boundary:**
   - All API requests and responses exchange raw integer cents (`*_cents: number`) and basis points (`*_basis_points: number`, where `10000 = 100.00%`).
   - User currency inputs (e.g., `"920,50"`) are parsed at the input boundary via `parseEuroToCents(input: string): number` into integer cents (`92050`), and formatted for display via `formatCentsToEuro(cents: number): string` (`"920,50 €"`) and `formatBasisPointsToPercent(bp: number): string` (`"50,00 %"`).
3. **Technology Stack & `shadcn-svelte` Preset (`web/components.json`):**
   - **Framework:** Svelte 5 (`^5.57.1`) + SvelteKit (`^3.0.0`) + `@sveltejs/adapter-cloudflare` + Vite 8 + Tailwind CSS v4 (`src/routes/layout.css`).
   - **`shadcn-svelte` CLI:** Install components via `npx shadcn-svelte@latest add <component>` inside `web/`.
   - **Preset:** `style: "mira"`, `baseColor: "zinc"`, `iconLibrary: "tabler"` (`@tabler/icons-svelte`).
   - **Subpath Aliases:** `#lib`, `#lib/components`, `#lib/components/ui`, `#lib/utils`, `#lib/hooks`.

---

## 4. User-Decided UX & Product Design

### 4.1. Operational Cockpit (`/`) — Entity → Property Rows with Inline CTAs
- **Structure (User Decision: Option B):** No separate inbox banner. The home screen is structured directly as a list of **Legal Entity cards**.
  - **Entity Header:** Shows entity name, legal form badge (`SCI IS`, `SCI IR`, `LMNP`, `Nom propre`), monthly rent collection summary (`Encaissé : X € / Y € attendus`), and an entity-level CTA if active leases are missing rent calls for the current month (`Générer appels du mois (N) →`).
  - **Property / Lot Rows inside each Entity:** Each property/lot displays its label, active tenant name, monthly rent + charges (`Loyer CC`), and **inline CTA buttons on the right side of the row** for every pending action:
    - If rent calls for the selected month are not yet generated for this lease: CTA `Générer appel →` → navigates to `/rents?entityId=<id>&leaseId=<id>&action=generate-calls`.
    - If a rent call is `PENDING` or `PARTIAL` and past due / in negative positions: CTA `⚠️ Relancer impayé →` → navigates to `/rents?entityId=<id>&leaseId=<id>&rentCallId=<id>&action=remind`.
    - If a rent call is `PENDING` or `PARTIAL`: CTA `Saisir loyer →` → navigates to `/rents?entityId=<id>&leaseId=<id>&rentCallId=<id>&action=pay`.
    - If the property/lot has open or in-progress incidents (`OPEN` / `IN_PROGRESS`): CTA `🔧 N Incident(s) →` → navigates to `/inspections?entityId=<id>&propertyId=<id>&incidentId=<id>&tab=incidents`.
    - If all rent calls for the month are `PAID` and no incidents are open: green status indicator `✓ À jour · Quittance disponible`.

### 4.2. Application Shell & Navigation (`(app)/+layout.svelte`)
- **Sidebar with Entity Switcher (User Decision: Option A):**
  - **Top Header of Sidebar:** `EntityWorkspaceSwitcher` dropdown allowing the user to select `"Toutes les entités"` (default consolidated view) or a specific Legal Entity. Selecting an entity updates the global workspace context (persisted in URL/storage) and filters all domain views.
  - **Main Domain Navigation:**
    1. `Cockpit Opérationnel` (`/`)
    2. `Loyers & Relances` (`/rents`) — with badge count of pending/overdue items
    3. `Incidents & États des lieux` (`/inspections`) — with badge count of open incidents
    4. `Biens & Lots` (`/properties`)
    5. `Baux & Locataires` (`/leases`)
    6. `Comptabilité & Fiscalité` (`/accounting`)
    7. `Entités & Associés` (`/entities`)
    8. `Administration & Accès` (`/admin`) — visible to `SUPER_ADMIN` (pending user approvals) and for managing personal MCP tokens
  - **Theme Switcher (User Decision — Official `shadcn-svelte` Dark Mode):** Uses [`mode-watcher`](https://www.shadcn-svelte.com/docs/dark-mode/svelte) with `<ModeWatcher />` mounted in the root `src/routes/+layout.svelte` and a `ThemeToggle.svelte` component using `toggleMode` / `setMode` / `resetMode` from `"mode-watcher"` (with `@tabler/icons-svelte` Sun/Moon icons).
  - **Sidebar Footer:** Logged-in user name, role badge (`SUPER_ADMIN` / `MANAGER`), and logout button.

### 4.3. Rents, Rent Calls & Reminders (`/rents`) — Unified Monthly Table + Contextual `Sheet`
- **Layout (User Decision: Option A):**
  - **Top Toolbar:** Month selector (`YYYY-MM`, defaulting to current month), quick status filter pills (`Tous`, `À encaisser`, `En retard / Impayés`, `Payés`), `Importer CSV bancaire` dialog trigger, `Dépôt d'espèces en banque` dialog trigger, and `Générer les appels du mois` button.
  - **Unified Monthly Table:** Lists every active lease / rent call for the selected month and entity context with: Property & Lot, Tenant, Rent Net + Charges + VAT = Total Due (`total_due_cents`), Remaining Due (`remaining_due_cents`), Due Date, Status badge (`En attente`, `Partiel`, `Payé`, `Impayé`), and row action button.
  - **Contextual Right-Hand `Sheet` (`RentActionSheet`):**
    - Opens automatically when arriving from a Cockpit CTA (`?leaseId=...&action=pay|remind`) or when clicking any row in the table.
    - **Payment Recording Section:** Pre-fills remaining due amount (`remaining_due_cents`), payment method (`BANK_TRANSFER`, `CHECK`, `DIRECT_DEBIT` via `POST /api/v1/payments/bank`, or `CASH` with receipt number via `POST /api/v1/payments/cash`), payment date, and reference. Submitting records the payment and refreshes the rent call status.
    - **Document & Receipt Section:** Allows generating/downloading the Rent Call notice (`RENT_CALL`) or Rent Receipt (`RECEIPT`) from templates (`POST /api/v1/documents/generate`) and listing attached lease documents (`GET /api/v1/leases/{id}/documents`).
    - **Overdue Reminder & GLI Claim Section:** Shows unpaid months from `GET /api/v1/entities/{id}/negative-positions`, allows generating reminder letters (`REMINDER` template) or assembling the 1-click **Dossier Sinistre GLI (`.zip`)** (`POST /api/v1/leases/{id}/gli-dossier`) with immediate download link.
  - **Collapsible Cash Journal Section:** Below or via a drawer toggle, displays the PCG 530 Cash Journal (`GET /api/v1/entities/{id}/cash-journal`).

### 4.4. Consistent Table/Cards + Contextual `Sheet` Across All Other Modules
- **Incidents & États des lieux (`/inspections`):**
  - Two view modes via segmented tabs:
    1. **Incidents:** Table of incidents (`GET /api/v1/incidents`) filterable by entity/property/status (`OPEN`, `IN_PROGRESS`, `RESOLVED`, `CLOSED`). Clicking a row or arriving from a Cockpit `🔧 Incident →` CTA opens the right-hand `IncidentSheet` to update status, insurance claim number, estimated cost (`estimated_cost_cents`), and observations (`PATCH /api/v1/incidents/{id}`), or create a new incident (`POST /api/v1/incidents`).
    2. **États des lieux (EDL) & Modèles:** Manage reusable room/element templates (`GET/POST /api/v1/inspection-templates`), create Move-In (`ENTRANT`) or Move-Out (`SORTANT`) inspections (`POST /api/v1/inspections`), edit room-by-room conditions (`NEW`, `GOOD`, `FAIR`, `POOR`, `DAMAGED`) and meter readings in an `InspectionSheet` (`PUT /api/v1/inspections/{id}/items`), and generate the comparative PDF report (`POST /api/v1/inspections/{id}/pdf`).
- **Biens & Lots (`/properties`):**
  - Lists properties and their nested lots for the selected entity (`GET /api/v1/entities/{id}/properties`).
  - Right-hand `Sheet` for creating a property (`POST /api/v1/entities/{id}/properties`, with `FIXED_ASSET` vs `MERCHANDISE`, `acquisition_cents`, `work_stock_cents`) and adding rentable lots (`POST /api/v1/properties/{id}/lots`, with surface $m^2$, ownership share / tantièmes, floor).
- **Baux & Locataires (`/leases`):**
  - Lists active/inactive leases (`GET /api/v1/leases`) and tenants (`GET /api/v1/tenants`).
  - Right-hand `Sheet` to create a tenant (`POST /api/v1/tenants`), archive a tenant for GDPR (`POST /api/v1/tenants/{id}/archive`), create a lease (`POST /api/v1/leases` with `RESIDENTIAL_EMPTY`, `RESIDENTIAL_FURNISHED`, `MOBILITY`, `COMMERCIAL_3_6_9`, VAT toggle, `vat_rate_basis_points`, `security_deposit_cents`, GLI policy, guarantor), and generate/upload lease documents (`/api/v1/documents/*`).
- **Entités & Associés (`/entities`):**
  - Lists legal entities (`GET /api/v1/entities`) and displays the detailed entity summary (`GET /api/v1/entities/{id}`) including total book value and CERFA 2059-F shareholders (`detention_basis_points`, `current_account_cents`, `remuneration_paid_cents`).
  - Right-hand `Sheet` to create a legal entity (`POST /api/v1/entities`) or upsert a shareholder (`POST /api/v1/entities/{id}/shareholders`).
- **Comptabilité & Fiscalité (`/accounting`):**
  - Requires an active entity selection (auto-selects the first entity if `"Toutes les entités"` is selected, with an explicit entity selector banner).
  - **Tab 1 — Comptes Annuels Interactifs & Liasse Fiscale (2072 / 2033):**
    - Calls `POST /api/v1/entities/{id}/statements/preview` with `{ fiscal_year, disabled_line_ids }`.
    - Displays summary cards (`Total Produits`, `Total Charges`, `Résultat Net`, `Capacité d'Autofinancement`, `Total Actif`, `Total Passif`) with $N$ vs $N-1$ variations.
    - Displays each statement line (`lines`) with a `Checkbox` (`shadcn-svelte`). Toggling a checkbox updates `disabledLineIds` (`$state`) and immediately recomputes the preview via the API without mutating the general ledger.
    - Actions: **Générer la Liasse & Comptes Annuels PDF** (`POST /api/v1/entities/{id}/statements/pdf`), **Exporter le FEC DGFiP (`.tsv`)** (`GET /api/v1/entities/{id}/fec`), and **Saisir / Importer des lignes de balance** (`PUT /api/v1/entities/{id}/statements/lines`).
  - **Tab 2 — Journal Comptable PCG:**
    - Lists balanced double-entry journal entries (`GET /api/v1/entities/{id}/journal-entries?fiscal_year=...`).
    - Right-hand `Sheet` to record a new balanced multi-line PCG journal entry (`POST /api/v1/entities/{id}/journal-entries`) with live debit/credit balance validation.
- **Authentification (`/login`, `/signup`) & Administration (`/admin`):**
  - `/login`: Email/password sign-in (`POST /api/v1/auth/login`). If the backend returns `403` (`account pending approval`), displays a clear informational alert.
  - `/signup`: Registration form (`POST /api/v1/auth/signup` returning `202 Accepted`), displaying a confirmation screen explaining that a `SUPER_ADMIN` must approve the account.
  - `/admin`:
    - **Jetons MCP (All authenticated users):** Generate personal MCP bearer tokens (`POST /api/v1/auth/mcp-tokens`) and copy the token once with a warning banner.
    - **Approbation des comptes (`SUPER_ADMIN` only):** List pending users (`GET /api/v1/admin/users/pending`) with `Approuver` (`POST /api/v1/admin/users/{id}/approve`) and `Rejeter` (`POST /api/v1/admin/users/{id}/reject`) buttons.

---

## 5. Technical Architecture & Builder Decisions (SOLID & Clean Code)

*(Marked as Builder Calls — sensible technical choices aligned with `AGENTS.md` and Svelte 5 best practices)*

### 5.1. Directory Structure (`web/src/`)
```text
web/src/
├── lib/
│   ├── core/
│   │   ├── api/
│   │   │   ├── http-client.ts        # Typed HttpClient interface + FetchHttpClient with automatic JWT Bearer & 401 token refresh
│   │   │   └── api-error.ts          # Standardized ApiError class mapping HTTP status & backend error messages
│   │   ├── formatters/
│   │   │   ├── money.ts              # formatCentsToEuro, parseEuroToCents, formatBasisPointsToPercent, parsePercentToBasisPoints
│   │   │   └── date.ts               # French locale date & YYYY-MM period formatters
│   │   └── workspace/
│   │       └── workspace.svelte.ts   # Reactive active entity context ("ALL" | entityId) + selected periodMonth
│   ├── modules/
│   │   ├── auth/
│   │   │   ├── types.ts              # UserDTO, AuthTokensResponse, LoginRequest, SignupRequest, MCPTokenResponse
│   │   │   ├── api.ts                # AuthApiService (login, signup, refresh, logout, createMcpToken, pendingUsers, approve, reject)
│   │   │   └── auth-store.svelte.ts  # Reactive AuthStore ($state user, accessToken, refreshToken, isAuthenticated, isSuperAdmin)
│   │   ├── portfolio/
│   │   │   ├── types.ts              # LegalEntityDTO, ShareholderDTO, PropertyDTO, LotDTO, EntityPortfolioSummaryDTO
│   │   │   └── api.ts                # PortfolioApiService
│   │   ├── lease/
│   │   │   ├── types.ts              # TenantDTO, LeaseDTO, RentCallDTO, CreateTenantCmd, CreateLeaseCmd
│   │   │   └── api.ts                # LeaseApiService
│   │   ├── payment/
│   │   │   ├── types.ts              # PaymentReceiptDTO, CashJournalEntryDTO, NegativePositionDTO, BatchImportResultDTO
│   │   │   └── api.ts                # PaymentApiService
│   │   ├── document/
│   │   │   ├── types.ts              # DocumentTemplateDTO, StoredFileDTO, GLIDossierArchiveDTO
│   │   │   └── api.ts                # DocumentApiService
│   │   ├── inspection/
│   │   │   ├── types.ts              # InspectionTemplateDTO, InspectionDTO, IncidentDTO, InspectionReportDTO
│   │   │   └── api.ts                # InspectionApiService
│   │   ├── accounting/
│   │   │   ├── types.ts              # JournalEntryDTO, ComputedFinancialStatementsDTO, AccountBalanceLineDTO
│   │   │   └── api.ts                # AccountingApiService
│   │   └── cockpit/
│   │       ├── types.ts              # EntityCockpitGroup, PropertyCockpitRow, PendingTaskAction
│   │       └── cockpit-builder.ts    # Pure domain function aggregating Entities, Properties, Leases, RentCalls, NegativePositions, and Incidents into Cockpit rows & CTAs
│   ├── components/
│   │   ├── ui/                       # Generated via `npx shadcn-svelte@latest add ...`
│   │   └── layout/
│   │       ├── AppSidebar.svelte
│   │       ├── EntityWorkspaceSwitcher.svelte
│   │       ├── ThemeToggle.svelte
│   │       └── PageHeader.svelte
│   └── utils.ts
└── routes/
    ├── layout.css
    ├── +layout.svelte                # Root layout (initializes theme & auth state)
    ├── (auth)/
    │   ├── login/+page.svelte
    │   └── signup/+page.svelte
    └── (app)/
        ├── +layout.svelte            # Authenticated shell with Sidebar, EntityWorkspaceSwitcher, ThemeToggle, and auth guard
        ├── +page.svelte              # Operational Cockpit (Option B: Entity -> Property rows with inline CTAs)
        ├── rents/+page.svelte        # Unified Monthly Table + RentActionSheet (Option A)
        ├── inspections/+page.svelte  # Incidents & Inspections + IncidentSheet / InspectionSheet
        ├── properties/+page.svelte   # Properties & Lots + PropertySheet / LotSheet
        ├── leases/+page.svelte       # Leases & Tenants + LeaseSheet / TenantSheet
        ├── entities/+page.svelte     # Legal Entities & Shareholders + EntitySheet / ShareholderSheet
        ├── accounting/+page.svelte   # Interactive Financial Statements & PCG Journal + JournalEntrySheet
        └── admin/+page.svelte        # User Approvals (SUPER_ADMIN) & MCP Token generator
```

### 5.2. Applying SOLID Principles in the Frontend
- **Single Responsibility Principle (SRP):**
  - Pure domain calculations (such as `buildCockpitGroups(...)` in `#lib/modules/cockpit/cockpit-builder.ts` and money/basis-point conversion in `#lib/core/formatters/money.ts`) have zero dependency on Svelte DOM or `fetch`.
  - API services (`#lib/modules/*/api.ts`) only serialize/deserialize DTOs and delegate transport to `HttpClient`.
  - `.svelte` components only bind reactive state and render `shadcn-svelte` UI primitives.
- **Open/Closed Principle (OCP):**
  - `cockpit-builder.ts` derives `PendingTaskAction[]` for each property row through modular rule evaluators (uncalled rent rule, overdue reminder rule, pending payment rule, open incident rule) so new operational signals can be added without modifying the Cockpit component.
- **Liskov Substitution Principle (LSP) & Dependency Inversion Principle (DIP):**
  - All module API services depend on the narrow `HttpClient` interface (`requestJson<T>`, `requestText`, `requestMultipart<T>`) rather than global `window.fetch`, enabling deterministic unit tests with an in-memory `MockHttpClient`.
- **Interface Segregation Principle (ISP):**
  - Presentational components and Sheets accept narrow, task-specific prop interfaces (`$props()`) rather than full module stores.

### 5.3. Development Proxy & Environment Configuration
- In `web/vite.config.ts`, configure `server.proxy` so `/api`, `/mcp`, and `/healthz` proxy to `process.env.PUBLIC_API_ORIGIN ?? 'http://localhost:8080'` during local development, while allowing `PUBLIC_API_BASE_URL` override in production if hosted on a separate origin.
- Disable SSR (`export const ssr = false;` in `src/routes/+layout.ts`) so the Cloudflare static/SPA bundle interacts cleanly with client-side JWT rotation and local storage.

### 5.4. Required `shadcn-svelte` Components & `mode-watcher`
- Install `mode-watcher` (`npm install mode-watcher`) following the official [shadcn-svelte Dark Mode guide](https://www.shadcn-svelte.com/docs/dark-mode/svelte): mount `<ModeWatcher />` in `src/routes/+layout.svelte` and use `toggleMode` / `setMode` / `resetMode` in `ThemeToggle.svelte`.
- Install and compose UI primitives using `npx shadcn-svelte@latest add`:
  `button`, `card`, `badge`, `table`, `tabs`, `sheet`, `dialog`, `field`, `input`, `label`, `select`, `checkbox`, `toggle-group`, `separator`, `dropdown-menu`, `alert`, `empty`, `skeleton`, `spinner`, `sidebar`, `tooltip`, `sonner`.
- Strictly follow `shadcn-svelte` composition rules (`Field.FieldGroup` + `Field.Field` for forms, `Select.Group` inside `Select.Content`, `DropdownMenu.Group` inside `DropdownMenu.Content`, `Empty.*` for empty states, `Spinner` with `data-icon="inline-start"` inside `Button` for loading states, `data-icon` on `@tabler/icons-svelte` icons without manual `size-*` overrides inside buttons/menus).

### 5.5. Verification & Testing Strategy
- Unit tests (`vitest`) for:
  - Monetary & basis-point formatters/parsers (`#lib/core/formatters/money.test.ts`).
  - Cockpit aggregation and CTA derivation logic (`#lib/modules/cockpit/cockpit-builder.test.ts`).
  - HTTP client token refresh & API services (`#lib/core/api/http-client.test.ts`).
- Static verification: `npm --prefix web run check` (`svelte-check`) and `npm --prefix web test` and `npm --prefix web run build` must pass with 0 errors and 0 warnings.
