# Architecture & Clean Code Rules (gestion-immo)

## 1. Language & Localization Boundary (100% English Code)
- **Strictly English in Code**: All source code identifiers (packages, structs, interfaces, functions, variables, constants, enum values, struct fields, JSON tags, TypeScript types/interfaces, and SQL table/column/constraint names) MUST be written **100% in English**. Never mix French and English (`Franglais`) in code or schemas.
- **French Only in Documents & UI**: French translations and domain labels (e.g., PCG/CERFA labels, lease types, inspection conditions, DGFiP FEC 18-column TSV headers) are strictly confined to:
  - Generated documents (`.docx` templates, HTML/PDF reports, normative FEC exports).
  - Frontend SvelteKit UI templates and display formatting helpers.

## 2. Monetary Arithmetic & Formatting Boundary
- **No Floating-Point Money**: Never use `float64` or `number` floats for monetary amounts or percentages. Always use `money.AmountCents` (`int64` euro cents) and `money.BasisPoints` (`int64` basis points where `10000 = 100.00%`).
- **Raw Integers in API DTOs**: REST API requests and responses (application-layer DTOs) must only transport raw integer fields (`*_cents`, `*_basis_points`). Never include `*_formatted` string fields in REST DTOs.
- **Formatting at the Edges**: Format amounts into human-readable French strings (`1 234,56 €`, `50,00 %`) exclusively in the SvelteKit Frontend (`$lib/utils.ts`) or inside backend document/PDF/FEC renderers.

## 3. Modular Monolith & Explicit Dependency Injection (`api/internal/modules/`)
- **Keep DI Simple & Explicit**: Do not use reflection-based DI frameworks or pass around a global `appContainer` god-object.
- **Module Struct Pattern**: Each bounded context module (`internal/modules/<name>/module.go`) must be a `Module` struct initialized via an explicit constructor `NewModule(...)` that accepts low-level infrastructure dependencies (`*pgxpool.Pool`, PDF engine, R2 storage, templating engine) or specific cross-module services.
- **Controlled Exposure**: Each `Module` struct wires its own repositories, domain/application services, and HTTP/MCP adapters internally, and exposes only required dependencies via explicit methods (e.g., `Service()`, `RegisterRoutes(...)`, `RegisterMCPTools(...)`).

## 4. CLI Architecture & Single Responsibility Principle (`api/cmd/`)
- **One Package Per Command**: Every CLI subcommand lives in its own package under `api/cmd/<command>/<command>.go` and exports `func Command() *cli.Command`. `api/cmd/main.go` only assembles the root `cli.Command` with `server.Command()`, `bootstrap.Command()`, `migrate.Command()`, and `token.Command()`.
- **Self-Contained Execution Units**: Each command must initialize **only** the dependencies it requires to run (e.g., `bootstrap` and `token` initialize only the `auth` module; `migrate` only runs database migrations; `server` initializes all modules).
- **Strict SRP for Migrations**: Database migrations must never be triggered by the HTTP server (`server` must not have `--auto-migrate` or `--migrate-path` flags). Migrations are executed exclusively via the dedicated `migrate` command.
- **Graceful Shutdown**: Wrap the command context with `ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` and `defer stop()` only in commands that manage long-running processes or interruptible I/O (such as `server`).

## 5. SOLID & Clean Code Principles
- **S — Single Responsibility Principle (SRP)**: Every package, struct, CLI command, and function must have a single reason to change (e.g., domain calculators never format HTML/PDF; `server` never runs DB migrations; HTTP handlers only decode/encode and delegate to application services).
- **O — Open/Closed Principle (OCP)**: Modules and services must be open for extension without modifying core domain logic (e.g., plugging in new storage/PDF/repository implementations or adding new MCP/HTTP adapters without altering domain entities).
- **L — Liskov Substitution Principle (LSP)**: Every repository or infrastructure implementation (e.g., `postgres` vs. `memory` repositories, R2 vs. in-memory object store) must honor the exact behavioral contract and error semantics of the domain interface it satisfies.
- **I — Interface Segregation Principle (ISP)**: Define small, consumer-specific Go interfaces where they are used (e.g., narrow cross-module reader/recorder interfaces like `CashJournalRecorder` or `LeaseLookup` rather than fat monolithic service interfaces).
- **D — Dependency Inversion Principle (DIP)**: High-level domain and application layers must never depend on low-level infrastructure packages (`pgxpool`, HTTP, S3/R2 SDKs). Both must depend on abstractions (interfaces) defined in the domain/application layer and wired at the module boundary (`module.go`).
- **Single Level of Abstraction & Small Functions**: Keep functions short, focused, and readable from top to bottom. Extract helper functions with clear, intention-revealing English names.
- **DRY (Don't Repeat Yourself)**: Avoid duplicating utility functions across modules or commands (share HTTP helpers in `internal/platform/httpx`, database connection helpers in `internal/platform/postgres`, and auth CLI wiring in `internal/modules/auth`).
- **KISS & YAGNI**: Favor explicit, straightforward code over speculative abstractions or over-engineered frameworks.
- **Zero Dead Code**: Remove unused flags, unreferenced types, and obsolete files immediately when refactoring. Always verify with `make -C api test && make -C api lint && make -C api build` and `npm --prefix web test && npm --prefix web run check`.
