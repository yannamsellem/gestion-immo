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
- **One Package Per Command**: Every CLI subcommand lives in its own package under `api/cmd/<command>/<command>.go` and exports `func Command() *cli.Command`. `api/cmd/main.go` only assembles the root `cli.Command` with `server.Command()`, `bootstrap.Command()`, and `token.Command()`.
- **Self-Contained Execution Units**: Each command must initialize **only** the dependencies it requires to run (e.g., `bootstrap` and `token` initialize only the `auth` module; `server` initializes all modules).
- **Strict SRP for Migrations & Keys**: Database migrations (`jackc/tern`) and cryptographic key generation (`step crypto`) must never be embedded in the Go application binary. Migrations are executed exclusively via the `tern` CLI locally (`Makefile`) or via the dedicated `migrate` initContainer in Docker/Kubernetes.
- **Graceful Shutdown**: Wrap the command context with `ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` and `defer stop()` only in commands that manage long-running processes or interruptible I/O (such as `server`).

## 5. SOLID & Clean Code Principles
- **S — Single Responsibility Principle (SRP)**: Every package, struct, CLI command, and function must have a single reason to change (e.g., domain calculators never format HTML/PDF; `server` never runs DB migrations; HTTP handlers only decode/encode and delegate to application services).
- **O — Open/Closed Principle (OCP)**: Modules and services must be open for extension without modifying core domain logic (e.g., plugging in new storage/PDF/repository implementations or adding new MCP/HTTP adapters without altering domain entities).
- **L — Liskov Substitution Principle (LSP)**: Every repository or infrastructure implementation (e.g., `postgres` vs. `memory` repositories, R2 vs. in-memory object store) must honor the exact behavioral contract and error semantics of the domain interface it satisfies.
- **I — Interface Segregation Principle (ISP)**: Define small, consumer-specific Go interfaces where they are used (e.g., narrow cross-module reader/recorder interfaces like `CashJournalRecorder` or `LeaseLookup` rather than fat monolithic service interfaces).
- **D — Dependency Inversion Principle (DIP)**: High-level domain and application layers must never depend on low-level infrastructure packages (`pgxpool`, HTTP, S3/R2 SDKs). Both must depend on abstractions (interfaces) defined in the domain/application layer and wired at the module boundary (`module.go`).
- **Single Level of Abstraction & Small Functions**: Keep functions short, focused, and readable from top to bottom. Extract helper functions with clear, intention-revealing English names.
- **DRY (Don't Repeat Yourself)**: Avoid duplicating utility functions across modules or commands (share HTTP helpers in `internal/platform/httpx`, database connection helpers in `internal/platform/postgres`, and auth CLI wiring in `internal/modules/auth`).
- **KISS, YAGNI & Vetted Libraries**: Favor explicit, idiomatic code (`make([]byte, n)` over clever fixed-array `[:]` slicing tricks) and mature reference libraries (e.g., `github.com/google/uuid` for RFC 9562 UUIDv7) over hand-rolling standard specifications or speculative abstractions.
- **Zero Dead Code & Mandatory Formatting/Linting**: Remove unused flags, unreferenced types, and obsolete files immediately when refactoring. Whenever any `.ts`, `.svelte`, or `.css` file in `web/` is created or modified, ALWAYS run `npm --prefix web run format` immediately afterward. Always verify with `make -C api test && make -C api lint && make -C api build` and `npm --prefix web run format && npm --prefix web run lint && npm --prefix web test && npm --prefix web run check`.

## 6. Identity & Secret Token Boundary
- **Native `UUIDv7` Entity IDs**: All persisted database entity primary and foreign keys use native PostgreSQL 18 `UUID` (`DEFAULT uuidv7()`) and canonical 36-character lowercase RFC 9562 UUIDv7 strings in Go (`internal/platform/id`) and OpenAPI (`format: uuid`). Never use TypeID/Stripe-style prefixes (`ent_...`, `prop_...`) on entity IDs.
- **Preserve Domain Slugs & Business Codes**: Keep natural/normative domain identifiers as `TEXT` (e.g., PCG/CERFA statement `line_id` slugs, JWT `kid`, cash receipt numbers `CAISSE-YYYYMMDD-XXXXXX`).
- **Selective Token Prefixing**: Prefix user-facing API/developer credentials (`mcp_live_<64-hex>`) for secret-scanning and visual identification; keep internal opaque session tokens (such as HTTP-only refresh tokens) as raw unprefixed 256-bit hex (`<64-hex>`).

## 7. Testing Discipline (Test Authored Code, Never Libraries)
- **Test Project Logic Only**: Every test must verify behavior authored in this repository—domain calculations, application invariants, boundary validations, security policies, custom parsers/formatters, or repository/HTTP/MCP wiring. Never write tests or assertions whose purpose is verifying third-party libraries or standard library primitives (`google/uuid` version bits, `crypto/rand`, `encoding/hex`, `urfave/cli`, `huma` schema defaults, `modelcontextprotocol/go-sdk` transports, JS `Map`/`Set`).
- **No Static Source-File Grepping or Tautologies**: Never test `.svelte` components or `.sql` migrations by reading source files as raw strings (`readFileSync`, `strings.Contains`, occurrence counts). Never write runtime tests that merely re-assert static enum/label constant tables already type-checked at compile time by `svelte-check` or `go build`.
- **Strict Layer Isolation & Zero Duplication**: Keep `domain` tests pure (never wire HTTP handlers, MCP registries, or cross-module application services inside `domain_test` packages) and avoid duplicating identical end-to-end scenarios across multiple packages.
