.PHONY: help setup dev build run test clean migrate seed fmt lint css test-docker-build test-docker-up test-docker-down test-docker-parallel test-docker-suite test-docker-logs test-docker-reset

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies and set up project
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy
	@echo "Installing Air for hot reload..."
	go install github.com/air-verse/air@latest
	@echo "Creating necessary directories..."
	mkdir -p data static/uploads tmp
	@echo "Setup complete!"

dev: ## Run development server with hot reload
	@echo "Starting development server with Air..."
	air

build: ## Build production binary
	@echo "Building production binary..."
	go build -o bin/dc-management-tool ./cmd/server
	@echo "Build complete: bin/dc-management-tool"

run: build ## Build and run production binary
	@echo "Running production binary..."
	./bin/dc-management-tool

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning up..."
	rm -rf tmp bin data/*.db static/uploads/*
	go clean
	@echo "Clean complete!"

migrate: ## Run database migrations
	@echo "Running migrations..."
	@go run cmd/server/main.go migrate || echo "Run 'make dev' to apply migrations automatically"

migrate-down: ## Rollback last migration (manual implementation needed)
	@echo "Migration rollback not yet implemented"
	@echo "To rollback, manually execute down.sql files"

seed: ## Seed database with test data
	@echo "Seeding database..."
	@sqlite3 data/dc_management.db < migrations/seed_data.sql
	@echo "Seed data inserted successfully!"

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	golangci-lint run

css: ## Build Tailwind CSS
	npm run css:build

restart: ## Stop, rebuild CSS + Go binary, and start the server
	@echo "Stopping any running server..."
	@-pkill -f 'air' 2>/dev/null || true
	@-pkill -f 'dc-management-tool' 2>/dev/null || true
	@sleep 1
	@echo "Rebuilding Tailwind CSS..."
	@npx tailwindcss -i static/css/tailwind-input.css -o static/css/tailwind-output.css 2>&1
	@echo "Building Go binary..."
	@go build -o bin/dc-management-tool ./cmd/server
	@echo "Starting server..."
	@./bin/dc-management-tool

## Docker-based parallel testing
test-docker-build: ## Build Docker test image
	@echo "Building Docker image..."
	@docker compose build

test-docker-up: ## Start 5 test containers (ports 8081-8085)
	@echo "Starting test containers..."
	@docker compose up -d --wait
	@echo "Containers ready on ports 8081-8085"

test-docker-down: ## Stop and remove test containers + volumes
	@echo "Stopping test containers..."
	@docker compose down -v

test-docker-parallel: ## Run all test plans in parallel
	@./scripts/run-parallel-tests.sh --all

test-docker-suite: ## Run single test suite: make test-docker-suite PLAN=phase-4-products-master PORT=8081
	@./scripts/run-suite.sh --plan testing-plans/$(PLAN).md --port $(PORT)

test-docker-logs: ## Tail test container logs
	@docker compose logs -f

test-docker-reset: ## Full teardown with orphan removal
	@echo "Full teardown..."
	@docker compose down -v --remove-orphans
	@echo "Done."

.DEFAULT_GOAL := help
