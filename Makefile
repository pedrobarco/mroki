HUB_DIR := web/mroki-hub
DEV_COMPOSE := build/dev/compose.yaml

.PHONY: help build test lint clean \
	api-build api-test api-test-verbose api-test-coverage api-fmt api-lint api-sqlc api-migrate api-clean \
	agent-build agent-test agent-clean \
	hub-build hub-test hub-test-ui hub-test-setup hub-screenshots hub-dev hub-install hub-preview hub-fmt hub-lint hub-clean \
	dev-up dev-down dev-reset

# ─── Global ──────────────────────────────────────────────────────────

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "  Global"
	@echo "  ────────────────────────────────────────"
	@echo "  build              Build all components"
	@echo "  test               Run all tests"
	@echo "  lint               Lint all components"
	@echo "  clean              Remove all build artifacts"
	@echo ""
	@echo "  API"
	@echo "  ────────────────────────────────────────"
	@echo "  api-build          Build mroki-api binary"
	@echo "  api-test           Run API tests"
	@echo "  api-test-verbose   Run API tests (verbose)"
	@echo "  api-test-coverage  Run API tests with coverage"
	@echo "  api-fmt            Format Go code"
	@echo "  api-lint           Run golangci-lint"
	@echo "  api-sqlc           Generate Go code from SQL"
	@echo "  api-migrate        Generate new migration file (usage: make api-migrate name=<name>)"
	@echo "  api-clean          Remove API build artifacts"
	@echo ""
	@echo "  Agent"
	@echo "  ────────────────────────────────────────"
	@echo "  agent-build        Build mroki-agent binary"
	@echo "  agent-test         Run Agent tests"
	@echo "  agent-clean        Remove Agent build artifacts"
	@echo ""
	@echo "  Hub"
	@echo "  ────────────────────────────────────────"
	@echo "  hub-build          Build hub for production"
	@echo "  hub-test           Run Playwright e2e tests"
	@echo "  hub-test-ui        Run e2e tests in UI mode"
	@echo "  hub-test-setup     Start backend for e2e tests"
	@echo "  hub-screenshots    Capture hub screenshots for docs"
	@echo "  hub-dev            Start hub dev server"
	@echo "  hub-install        Install hub dependencies"
	@echo "  hub-preview        Preview production build"
	@echo "  hub-fmt            Format hub code"
	@echo "  hub-lint           Lint hub code"
	@echo "  hub-clean          Remove hub build artifacts"
	@echo ""
	@echo "  Dev Stack"
	@echo "  ────────────────────────────────────────"
	@echo "  dev-up             Start dev stack (db + api + agent)"
	@echo "  dev-down           Stop dev stack"
	@echo "  dev-reset          Reset dev stack (destroy + recreate)"

build: api-build agent-build hub-build

test: api-test agent-test hub-test

lint: api-lint hub-lint

clean: api-clean agent-clean hub-clean

# ─── API ─────────────────────────────────────────────────────────────

api-build:
	@echo "Building mroki-api..."
	@mkdir -p bin
	go build -o bin/mroki-api ./cmd/mroki-api

api-test:
	@echo "Running API tests..."
	go test ./cmd/mroki-api/... ./internal/... ./pkg/...

api-test-verbose:
	@echo "Running API tests (verbose)..."
	go test -v ./cmd/mroki-api/... ./internal/... ./pkg/...

api-test-coverage:
	@echo "Running API tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./cmd/mroki-api/... ./internal/... ./pkg/...
	@echo "Coverage report: coverage.out"
	@echo "View in browser: go tool cover -html=coverage.out"

api-fmt:
	@echo "Formatting Go code..."
	go fmt ./...

api-lint:
	@echo "Running golangci-lint..."
	golangci-lint run

api-sqlc:
	@echo "Generating Go code from SQL..."
	go tool sqlc generate

api-migrate:
ifndef name
	$(error name is required. Usage: make api-migrate name=<migration_name>)
endif
	@echo "Starting dev database..."
	@docker run --rm -d --name atlas-dev-db \
		-e POSTGRES_PASSWORD=pass -e POSTGRES_DB=test \
		-p 5433:5432 postgres:15 > /dev/null
	@until docker exec atlas-dev-db pg_isready -U postgres > /dev/null 2>&1; do sleep 0.5; done
	@echo "Generating migration: $(name)..."
	@go run -mod=mod ent/migrate/main.go $(name); \
		EXIT=$$?; \
		docker stop atlas-dev-db > /dev/null 2>&1; \
		exit $$EXIT
	@echo "Migration generated in ent/migrate/migrations/"

api-clean:
	@echo "Cleaning API build artifacts..."
	rm -rf bin/mroki-api
	rm -f coverage.out

# ─── Agent ───────────────────────────────────────────────────────────

agent-build:
	@echo "Building mroki-agent..."
	@mkdir -p bin
	go build -o bin/mroki-agent ./cmd/mroki-agent

agent-test:
	@echo "Running Agent tests..."
	go test ./cmd/mroki-agent/...

agent-clean:
	@echo "Cleaning Agent build artifacts..."
	rm -rf bin/mroki-agent

# ─── Hub ─────────────────────────────────────────────────────────────

hub-build:
	@echo "Building hub..."
	cd $(HUB_DIR) && pnpm build

hub-test:
	@echo "Running hub e2e tests..."
	cd $(HUB_DIR) && pnpm test:e2e

hub-test-ui:
	@echo "Running hub e2e tests (UI mode)..."
	cd $(HUB_DIR) && pnpm test:e2e:ui

hub-test-setup:
	@echo "Starting backend for e2e tests..."
	cd $(HUB_DIR) && pnpm test:e2e:setup

hub-screenshots:
	@echo "Capturing hub screenshots..."
	cd $(HUB_DIR) && pnpm screenshots

hub-dev:
	@echo "Starting hub dev server..."
	cd $(HUB_DIR) && pnpm dev

hub-install:
	@echo "Installing hub dependencies..."
	cd $(HUB_DIR) && pnpm install

hub-preview:
	@echo "Previewing hub build..."
	cd $(HUB_DIR) && pnpm preview

hub-fmt:
	@echo "Formatting hub..."
	cd $(HUB_DIR) && pnpm format

hub-lint:
	@echo "Linting hub..."
	cd $(HUB_DIR) && pnpm lint

hub-clean:
	@echo "Cleaning hub build artifacts..."
	rm -rf $(HUB_DIR)/dist
	rm -rf $(HUB_DIR)/playwright-report
	rm -rf $(HUB_DIR)/e2e/test-results

# ─── Dev Stack ───────────────────────────────────────────────────────

dev-up:
	@echo "Starting dev stack..."
	docker compose -f $(DEV_COMPOSE) up -d --build --wait

dev-down:
	@echo "Stopping dev stack..."
	docker compose -f $(DEV_COMPOSE) down

dev-reset:
	@echo "Resetting dev stack..."
	docker compose -f $(DEV_COMPOSE) down -v
	docker compose -f $(DEV_COMPOSE) up -d --build --wait
