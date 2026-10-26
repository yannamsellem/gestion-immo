BINARY_NAME := gestion-immo
BINARY_PATH := bin/$(BINARY_NAME)
COMPOSE_FILE := docker-compose.yml

.DEFAULT_GOAL := help

.PHONY: help build dev test test-race lint migrate-up migrate-down migrate-create keys bootstrap token docker-up docker-down

help: ## Display this help message
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

build: ## Build the gestion-immo binary into bin/gestion-immo
	@mkdir -p bin
	go build -o $(BINARY_PATH) ./cmd

dev: ## Run the gestion-immo server in development mode
	go run ./cmd server

test: ## Run all Go unit and integration tests
	go test ./...

test-race: ## Run all Go tests with the race detector enabled
	go test -race ./...

lint: ## Run go vet static analysis across all packages
	go vet ./...

migrate-up: ## Apply all pending PostgreSQL migrations
	go run ./cmd migrate up

migrate-down: ## Roll back the last PostgreSQL migration
	go run ./cmd migrate down

migrate-create: ## Create a new SQL migration pair (usage: make migrate-create NAME=my_migration)
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required (e.g. make migrate-create NAME=add_table)"; exit 1; fi
	@mkdir -p migrations
	@NEXT=$$(printf "%06d" $$(( $$(ls migrations/*.up.sql 2>/dev/null | wc -l) + 1 ))); \
	touch "migrations/$${NEXT}_$(NAME).up.sql" "migrations/$${NEXT}_$(NAME).down.sql"; \
	echo "Created migrations/$${NEXT}_$(NAME).up.sql and .down.sql"

keys: ## Generate a local Ed25519 JWT signing keypair into ./keys
	go run ./cmd token keygen

bootstrap: ## Bootstrap the initial SUPER_ADMIN account
	go run ./cmd bootstrap super-admin

token: ## Generate a new SHA-256 hashed MCP Bearer token
	go run ./cmd token create

docker-up: ## Start all Docker Compose services in detached mode
	docker compose -f $(COMPOSE_FILE) up -d --build

docker-down: ## Stop and remove Docker Compose containers
	docker compose -f $(COMPOSE_FILE) down
