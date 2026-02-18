.PHONY: help setup dev build run test clean migrate migrate-status migrate-down seed fmt lint css restart

DATABASE_PATH ?= ./data/dc_management.db

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
	@echo "Installing Goose migration CLI..."
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "Creating necessary directories..."
	mkdir -p data static/uploads tmp internal/migrations
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

migrate: ## Run database migrations (via goose)
	@echo "Running migrations..."
	@goose -dir internal/migrations sqlite3 $(DATABASE_PATH) up

migrate-status: ## Show migration status
	@goose -dir internal/migrations sqlite3 $(DATABASE_PATH) status

migrate-down: ## Rollback last migration
	@goose -dir internal/migrations sqlite3 $(DATABASE_PATH) down

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

.DEFAULT_GOAL := help
