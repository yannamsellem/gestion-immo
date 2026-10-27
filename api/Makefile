# ------------------------------------------------------------------------------
# Configuration & Paths
# ------------------------------------------------------------------------------

BINARY_NAME    := gestion-immo
BINARY_PATH    := $(CURDIR)/bin/$(BINARY_NAME)
COMPOSE_FILE   := $(CURDIR)/docker-compose.yml
MIGRATIONS_DIR := $(CURDIR)/migrations
TERN_CONF      := $(MIGRATIONS_DIR)/tern.conf
KEYS_DIR       := $(CURDIR)/keys

.DEFAULT_GOAL := help

.PHONY: help build dev test test-race lint migrate-up migrate-down migrate-status migrate-create generate-keys bootstrap token docker-up docker-down

# ------------------------------------------------------------------------------
# General & Help
# ------------------------------------------------------------------------------

help: ## Display this help message
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ------------------------------------------------------------------------------
# Build, Run & Quality
# ------------------------------------------------------------------------------

build: ## Build the gestion-immo binary into bin/gestion-immo
	@mkdir -p $(dir $(BINARY_PATH))
	go build -o $(BINARY_PATH) ./cmd

dev: ## Run the gestion-immo server in development mode
	go run ./cmd server

test: ## Run all Go unit and integration tests
	go test ./...

test-race: ## Run all Go tests with the race detector enabled
	go test -race ./...

lint: ## Run go vet static analysis across all packages
	go vet ./...

# ------------------------------------------------------------------------------
# Database Migrations (jackc/tern)
# ------------------------------------------------------------------------------

migrate-up: ## Apply all pending PostgreSQL migrations using tern
	@tern migrate -m $(MIGRATIONS_DIR) -c $(TERN_CONF)

migrate-down: ## Roll back the last PostgreSQL migration using tern
	@tern migrate -m $(MIGRATIONS_DIR) -c $(TERN_CONF) --destination -1

migrate-status: ## Display PostgreSQL migration status using tern
	@tern status -m $(MIGRATIONS_DIR) -c $(TERN_CONF)

migrate-create: ## Create a new tern SQL migration (usage: make migrate-create NAME=my_migration)
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required (e.g. make migrate-create NAME=add_table)"; exit 1; fi
	@tern new -m $(MIGRATIONS_DIR) $(NAME)

# ------------------------------------------------------------------------------
# Cryptographic Keys (Smallstep / JWKS)
# ------------------------------------------------------------------------------

generate-keys: ## Generate Ed25519 private key and static JWKS JSON using Smallstep
	@mkdir -p $(KEYS_DIR)
	@step crypto keypair $(KEYS_DIR)/public.pem $(KEYS_DIR)/private.pem --kty OKP --crv Ed25519 --no-password --insecure --force
	@step crypto jwk create $(KEYS_DIR)/jwk.pub.json $(KEYS_DIR)/jwk.priv.json --from-pem $(KEYS_DIR)/private.pem --alg EdDSA --use sig --no-password --insecure --force
	@cat $(KEYS_DIR)/jwk.pub.json | step crypto jwk keyset add $(KEYS_DIR)/jwks.json
	@rm -f $(KEYS_DIR)/jwk.pub.json $(KEYS_DIR)/jwk.priv.json $(KEYS_DIR)/public.pem
	@echo "Keys successfully generated in $(KEYS_DIR)/"

# ------------------------------------------------------------------------------
# Administration & MCP Tokens
# ------------------------------------------------------------------------------

bootstrap: ## Bootstrap the initial SUPER_ADMIN account
	go run ./cmd bootstrap super-admin

token: ## Generate a new SHA-256 hashed MCP Bearer token
	go run ./cmd token create

# ------------------------------------------------------------------------------
# Docker Compose
# ------------------------------------------------------------------------------

docker-up: ## Start all Docker Compose services in detached mode
	docker compose -f $(COMPOSE_FILE) up -d --build

docker-down: ## Stop and remove Docker Compose containers
	docker compose -f $(COMPOSE_FILE) down
