.PHONY: help setup dev build run test clean migrate seed fmt lint

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

.DEFAULT_GOAL := help
