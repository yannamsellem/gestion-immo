# Plan 001: Fix Makefile migration generator, local onboarding, backup script pipeline, and go.mod dependencies

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: If git commits exist, run `git diff --stat HEAD -- api/Makefile api/Dockerfile api/docker-compose.yml api/scripts/backup-to-r2.sh api/internal/platform/config/config.go api/cmd/token/token.go api/internal/platform/postgres/migrator.go api/go.mod`. Compare the "Current state" excerpts against the live code before proceeding; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx | bug | security | migration
- **Covers findings**: `DX-01`, `DX-02`, `BUG-03`, `DEP-01`
- **Planned at**: commit `uncommitted-initial-tree` (no `HEAD` commit yet), 2026-10-26

## Why this matters

Currently, running `make migrate-create NAME=foo` inside `api/` expands `$( ls ...)` as an empty GNU Make macro and always generates `migrations/000001_foo.up.sql`, colliding with the initial schema migration. New developers or agents also cannot run the stack locally because `api/.env.example` is missing, `postgres` in `api/docker-compose.yml` exposes no host port and runs no migrations at startup, and `JWT_KEYS_DIR` defaults to `/app/keys` with no keygen helper. Furthermore, `api/scripts/backup-to-r2.sh` pipes `pg_dump | gzip -9` under POSIX `sh` without `pipefail` or a cleanup `trap` (so a failed `pg_dump` silently uploads a 20-byte empty `.sql.gz` backup to Cloudflare R2 and leaks `/tmp` files), and `api/go.mod` marks 7 directly imported packages as `// indirect` while shipping `github.com/lib/pq` alongside `jackc/pgx/v5`. Fixing this baseline unblocks every subsequent schema and integration plan.

## Current state

- `api/Makefile` — developer task runner with unescaped `$( ls ...)` on line 37 and `server run` on line 17:
  ```makefile
  # api/Makefile:16-18
  dev: ## Run the gestion-immo server in development mode
  	go run ./cmd server run

  # api/Makefile:34-40
  migrate-create: ## Create a new SQL migration pair (usage: make migrate-create NAME=my_migration)
  	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required (e.g. make migrate-create NAME=add_table)"; exit 1; fi
  	@mkdir -p migrations
  	@NEXT=$$(printf "%06d" $$(($( ls migrations/*.up.sql 2>/dev/null | wc -l ) + 1))); \
  	touch "migrations/$${NEXT}_$(NAME).up.sql" "migrations/$${NEXT}_$(NAME).down.sql"; \
  	echo "Created migrations/$${NEXT}_$(NAME).up.sql and .down.sql"
  ```
- `api/Dockerfile` — container build definition passing `CMD ["server", "run"]` at line 29:
  ```dockerfile
  # api/Dockerfile:28-29
  ENTRYPOINT ["/app/gestion-immo"]
  CMD ["server", "run"]
  ```
- `api/docker-compose.yml` — compose stack missing host port mapping on `postgres`, missing a migration init step, and running `apk add --no-cache aws-cli >/dev/null 2>&1 || true` as `root` in `pg-backup`:
  ```yaml
  # api/docker-compose.yml:60-76
    pg-backup:
      image: postgres:16-alpine
      restart: unless-stopped
      depends_on:
        postgres:
          condition: service_healthy
      ...
      volumes:
        - ./scripts/backup-to-r2.sh:/usr/local/bin/backup-to-r2.sh:ro
      entrypoint: ["/bin/sh", "-c", "apk add --no-cache aws-cli >/dev/null 2>&1 || true; while true; do /usr/local/bin/backup-to-r2.sh; sleep 86400; done"]
  ```
- `api/scripts/backup-to-r2.sh` — daily R2 backup script masking `pg_dump` exit status in a pipeline and lacking `umask 077` / `trap`:
  ```sh
  # api/scripts/backup-to-r2.sh:10-23
  TIMESTAMP="$(date -u +%Y-%m-%dT%H-%M-%SZ)"
  BACKUP_FILE="/tmp/gestion-immo-${TIMESTAMP}.sql.gz"
  OBJECT_KEY="daily/gestion-immo-${TIMESTAMP}.sql.gz"

  echo "[backup] creating compressed pg_dump archive ${OBJECT_KEY}..."
  pg_dump "${DATABASE_URL}" | gzip -9 > "${BACKUP_FILE}"

  AWS_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID}" \
  AWS_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY}" \
  AWS_DEFAULT_REGION="auto" \
  aws s3 cp "${BACKUP_FILE}" "s3://${R2_BACKUP_BUCKET_NAME}/${OBJECT_KEY}" --endpoint-url "${R2_ENDPOINT}"

  rm -f "${BACKUP_FILE}"
  ```
- `api/internal/platform/config/config.go` — defaults `JWT_KEYS_DIR` to `"/app/keys"` (lines 66–69).
- `api/cmd/token/token.go` — defines `token create` (lines 17–60), while `domain.GenerateTokenAuthority` and `TokenAuthority.EncodePEM()` already exist in `api/internal/modules/auth/domain/token_authority.go`.
- `api/internal/platform/postgres/migrator.go` — blank-imports `_ "github.com/golang-migrate/migrate/v4/database/postgres"` (line 10), pulling in `github.com/lib/pq`.

## Commands you will need

| Purpose | Command | Expected on success |
| ------- | ------- | ------------------- |
| Vet | `cd api && go vet ./...` | exit 0, no output |
| Test | `cd api && go test ./...` | exit 0, all packages `ok` |
| Tidy check | `cd api && go mod tidy` | exit 0 |

## Scope

**In scope** (the only files you should modify or create):

- `api/Makefile`
- `api/Dockerfile`
- `api/docker-compose.yml`
- `api/.env.example` (create)
- `api/scripts/backup-to-r2.sh`
- `api/internal/platform/config/config.go`
- `api/internal/platform/config/config_test.go`
- `api/cmd/token/token.go`
- `api/cmd/main_test.go`
- `api/internal/platform/postgres/migrator.go`
- `api/go.mod`
- `api/go.sum`

**Out of scope** (do NOT touch):

- Any SQL files under `api/migrations/` (handled in Plan 007).
- Any domain or application service files under `api/internal/modules/`.
- Anything in `web/` or `infra/`.

## Git workflow

- Branch: `advisor/001-dx-makefile-compose-and-backup-hardening`
- Commit message style: `chore(api): fix makefile, local dev setup, backup script, and go.mod`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Fix `api/Makefile`, `api/Dockerfile`, and create `api/.env.example` (`DX-01`, `DX-02`)

1. In `api/Makefile`:
   - Change line 17 from `go run ./cmd server run` to `go run ./cmd server`.
   - Fix line 37 in `migrate-create` to escape the inner `$$(ls ...)` subshell so Make does not swallow it:
     ```makefile
     migrate-create: ## Create a new SQL migration pair (usage: make migrate-create NAME=my_migration)
     	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required (e.g. make migrate-create NAME=add_table)"; exit 1; fi
     	@mkdir -p migrations
     	@NEXT=$$(printf "%06d" $$(( $$(ls migrations/*.up.sql 2>/dev/null | wc -l) + 1 ))); \
     	touch "migrations/$${NEXT}_$(NAME).up.sql" "migrations/$${NEXT}_$(NAME).down.sql"; \
     	echo "Created migrations/$${NEXT}_$(NAME).up.sql and .down.sql"
     ```
   - Add convenience targets `keys` (`go run ./cmd token keygen`), `bootstrap` (`go run ./cmd bootstrap super-admin`), and `token` (`go run ./cmd token create`), and add them to `.PHONY`.
2. In `api/Dockerfile:29`, change `CMD ["server", "run"]` to `CMD ["server"]`.
3. Create `api/.env.example` containing documented dummy placeholders for `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `DATABASE_URL`, `JWT_KEYS_DIR=./keys`, `JWT_KID=local-dev-key`, `R2_ENDPOINT`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET_NAME`, `R2_BACKUP_BUCKET_NAME`, `GOTENBERG_URL=http://localhost:3000`, `ENABLE_OPEN_BANKING=false`, `PORT=8080`, `CLOUDFLARE_TUNNEL_TOKEN`, `SUPER_ADMIN_EMAIL`, and `SUPER_ADMIN_PASSWORD` (no real secrets).

**Verify**: `sh -n api/scripts/backup-to-r2.sh` → exit 0.

### Step 2: Add `token keygen` CLI subcommand and local `JWT_KEYS_DIR` fallback (`DX-02`)

1. In `api/cmd/token/token.go`, add a `keygen` subcommand under `token` with `--out-dir` (default `"./keys"`, env `JWT_KEYS_DIR`) and `--kid` (default `"default"`, env `JWT_KID`) that calls `domain.GenerateTokenAuthority(kid)` and `authority.EncodePEM()`, creates `--out-dir` with mode `0700`, and writes `private.pem` (`0600`) and `public.pem` (`0644`).
2. In `api/internal/platform/config/config.go:66-69`, keep `"/app/keys"` as the default `JWTKeysDir` when `JWT_KEYS_DIR` is unset (preserving existing `config_test.go` expectations), and document `JWT_KEYS_DIR=./keys` in `api/.env.example`. Add a test in `api/cmd/main_test.go` testing `token keygen --out-dir < t.TempDir() >`.

**Verify**: `cd api && go test ./cmd/... ./internal/platform/config/...` → exit 0.

### Step 3: Harden `api/scripts/backup-to-r2.sh` and `api/docker-compose.yml` (`BUG-03`, `DX-02`)

1. In `api/scripts/backup-to-r2.sh`:
   - Add `umask 077` before creating temporary files.
   - Register `trap 'rm -f "${BACKUP_FILE}"' EXIT INT TERM`.
   - Replace `pg_dump "${DATABASE_URL}" | gzip -9 > "${BACKUP_FILE}"` with `pg_dump --no-owner --no-acl -Z 9 -f "${BACKUP_FILE}" "${DATABASE_URL}"` (so `pg_dump` exit code is checked directly under `set -eu` without a pipe masking failure), followed by `[ -s "${BACKUP_FILE}" ] || { echo "[backup] error: backup file is empty" >&2; exit 1; }`.
2. In `api/docker-compose.yml`:
   - Under `postgres`, add `ports: ["127.0.0.1:${POSTGRES_PORT:-5432}:5432"]`.
   - Add a `migrate` service using the same image/build as `gestion-immo` with `command: ["migrate", "up"]`, `restart: "no"`, depending on `postgres: service_healthy`, and make `gestion-immo` depend on `migrate: condition: service_completed_successfully`.
   - In `pg-backup`, remove `|| true` from `apk add --no-cache aws-cli`.

**Verify**: `sh -n api/scripts/backup-to-r2.sh` → exit 0.

### Step 4: Switch `migrator.go` to `pgx/v5` and tidy `api/go.mod` (`DEP-01`)

1. In `api/internal/platform/postgres/migrator.go`:
   - Replace `_ "github.com/golang-migrate/migrate/v4/database/postgres"` with `_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"` (or use `database/pgx/v5` with `pgx5://` scheme conversion in `RunMigrations`: replace leading `postgres://` or `postgresql://` with `pgx5://` before passing to `migrate.New`).
2. Run `cd api && go mod tidy` so that direct imports (`aws-sdk-go-v2`, `golang-jwt/jwt/v5`, `modelcontextprotocol/go-sdk`) move into the direct `require` block and `github.com/lib/pq` is removed.

**Verify**: `cd api && go vet ./... && go test ./...` → exit 0.

## Test plan

- Add `TestTokenKeygenCommand` in `api/cmd/main_test.go` verifying that `token keygen --out-dir <tempDir>` creates non-empty `private.pem` (`0600`) and `public.pem` that `domain.LoadTokenAuthorityFromDir` can load.
- Run `cd api && go test -v ./cmd/... ./internal/platform/...` → all pass.

## Done criteria

- [ ] `cd api && go vet ./...` exits 0
- [ ] `cd api && go test ./...` exits 0
- [ ] `sh -n api/scripts/backup-to-r2.sh` exits 0
- [ ] `grep -c "lib/pq" api/go.mod` outputs `0`
- [ ] `test -f api/.env.example` exits 0
- [ ] No files outside the in-scope list are modified
- [ ] `plans/README.md` status row updated

## STOP conditions

- If `go mod tidy` requires network access to fetch new module hashes and fails inside the sandbox, keep `_ "github.com/golang-migrate/migrate/v4/database/postgres"` and only promote already-cached packages in `go.mod`, or report the sandbox network block.
- If any file in "Current state" has drifted from the excerpts above.

## Maintenance notes

- `api/Makefile`'s `migrate-create` target is used in Plan 007 to create `000007_schema_constraints_and_indexes.{up,down}.sql`.
- `api/.env.example` must be kept in sync whenever new environment variables are added to `api/internal/platform/config/config.go`.
