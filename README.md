# Gestion Immo

Multi-entity real estate portfolio, lease, property inspection, and accounting management platform (SCI IR/IS, SARL IS, Family SARL, SAS, Real Estate Dealer, Sole Proprietorship & Joint Ownership), featuring a native **Streamable HTTP MCP (Model Context Protocol)** server.

---

## Key Features

- **Multi-Entity Portfolio & Shareholders**: Manage legal structures, share capital breakdown, and shareholder current accounts (CERFA 2059-F), with strict accounting separation between fixed assets (PCG 211/213) and real estate dealer inventory (PCG 355/335).
- **Leases, Rent Billing & Arrears**: Residential leases (VAT-exempt) and commercial leases (optional VAT), monthly rent call generation, FIFO payment allocation for bank transfers (PCG 512) and cash payments (cash journal PCG 530), and automatic detection of negative tenant positions.
- **1-Click GLI Claim Dossier**: Automated assembly of a complete ZIP archive stored on Cloudflare R2 containing the signed lease, move-in inspection report, tenant ledger, formal notices, and GLI insurance claim schedule.
- **Move-In / Move-Out Property Inspections**: Room-by-room condition tracking with Cloudflare R2 photo evidence, automated wear/damage comparison between move-in and move-out inspections, and PDF report generation.
- **Interactive Annual Accounts & DGFiP FEC Export**: Real-time interactive preview of the Balance Sheet (2033-A), Income Statement (2033-B), and Intermediate Management Balances (SIG) with on-the-fly line toggling, SGCAM/CERFA PDF report generation, and normative 18-column DGFiP FEC export (Art. A.47 A-1 LPF).
- **Native MCP Server (`/mcp`)**: Exposes domain tools over the Model Context Protocol (Streamable HTTP), secured with SHA-256 hashed Bearer tokens and OAuth 2.1 Protected Resource Metadata discovery (`/.well-known/oauth-protected-resource`).

---

## Tech Stack

### Backend (`api/`)
- **Language & Architecture**: Go 1.27 — Modular Monolith built around Clean Architecture and SOLID principles (`domain`, `application`, `infrastructure`, `interfaces`).
- **Database**: PostgreSQL 18 (`jackc/pgx/v5`) with native `UUIDv7` primary keys and transactional DDL schema migrations managed by **`jackc/tern`** (locally via CLI and in production via a dedicated `migrate` initContainer).
- **CLI**: `urfave/cli/v3` with self-contained subcommands (`server`, `bootstrap`, `token`).
- **Security & Authentication**: Argon2id password hashing, asymmetric Ed25519 JWTs (generated via **Smallstep `step`**), `SUPER_ADMIN` user approval workflow, and SHA-256 hashed MCP Bearer tokens.
- **Document Engine & Object Storage**: `.docx` templating and HTML-to-PDF rendering powered by **Gotenberg 8**; S3-compatible object storage on **Cloudflare R2**.
- **Monetary Precision**: Integer-only arithmetic in euro cents (`int64`) and basis points (`10000 = 100.00%`), with zero floating-point calculations.

### Frontend (`web/`)
- **Framework**: SvelteKit 2 (SSR) & TypeScript.
- **UI & Design System**: Tailwind CSS & `shadcn-svelte` components.
- **Hosting**: Cloudflare Pages.

### Infrastructure (`infra/terraform/`)
- **IaC**: Terraform (Cloudflare provider) provisioning Cloudflare R2 buckets (application documents and encrypted daily PostgreSQL backups), Cloudflare Zero Trust Tunnel (`cloudflared`), DNS records, and the Cloudflare Pages project.
- **Containerization**: Docker & Docker Compose (`postgres`, `migrate` initContainer, `gotenberg`, `gestion-immo`, `cloudflared`, `pg-backup`).

---

## Getting Started

### 1. Prerequisites
- **Go** `>= 1.27`
- **Node.js** `>= 22` & **npm**
- **Smallstep CLI** (`step`) for cryptographic key and JWKS generation
- **Tern CLI** (`jackc/tern/v2`) for PostgreSQL schema migrations
- **PostgreSQL** `18` & **Gotenberg** `8` (or **Docker Compose**; if resetting a local pre-18 Docker volume, run `docker compose -f api/docker-compose.yml down -v` first)

### 2. Cryptographic Keys & Database Migrations
Generate the Ed25519 signing keypair and static JWKS using Smallstep, then apply SQL migrations via `tern`:

```bash
# Generate Ed25519 private key and static JWKS JSON in api/keys/
make -C api generate-keys

# Apply pending SQL migrations (jackc/tern)
make -C api migrate-up

# Check migration status or roll back the last migration
make -C api migrate-status
make -C api migrate-down
```

### 3. Super Admin Bootstrap & MCP Tokens
```bash
# Create or update the SUPER_ADMIN account
go run ./api/cmd bootstrap super-admin \
  --email "admin@example.com" \
  --password "your-strong-password"

# List and approve pending user accounts
go run ./api/cmd bootstrap pending
go run ./api/cmd bootstrap approve --email "manager@example.com"

# Generate a Bearer token for an MCP client (e.g. Antigravity)
go run ./api/cmd token create --email "admin@example.com" --label "antigravity-mcp"
```

### 4. Run in Development
```bash
# Start the HTTP REST API & Streamable HTTP MCP server (/mcp) on port 8080
make -C api dev

# Start the SvelteKit frontend
npm --prefix web install
npm --prefix web run dev
```

---

## Testing & Code Quality

```bash
# Go Backend: unit tests, static analysis (go vet), and build
make -C api test
make -C api lint
make -C api build

# SvelteKit Frontend: unit tests and API/SSR contract checks
npm --prefix web test
npm --prefix web run check
```
