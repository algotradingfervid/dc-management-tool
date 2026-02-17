# Plan: Docker-Based Parallel Playwright-CLI Testing

## Context
The project has 14 detailed testing plans in `testing-plans/` (phases 1-15) written for `playwright-cli`. Currently tests run sequentially against a single `localhost:8080` instance, causing shared state conflicts if multiple agents test simultaneously. The goal is to spin up isolated Docker containers so multiple agents can run different phase test plans in parallel, each against its own app instance.

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Host Machine                                    │
│                                                  │
│  Agent 1 ──playwright-cli──► :8081 (container 1) │
│  Agent 2 ──playwright-cli──► :8082 (container 2) │
│  Agent 3 ──playwright-cli──► :8083 (container 3) │
│  Agent 4 ──playwright-cli──► :8084 (container 4) │
│  ...                                             │
└─────────────────────────────────────────────────┘
```

Each container: own SQLite DB, own migrations, own seed data. Complete isolation.

## Files to Create

### 1. `Dockerfile` (multi-stage build)
- **Builder stage**: `golang:1.25-bookworm` (has gcc for CGO/sqlite3)
  - Copy go.mod/go.sum, `go mod download`
  - Copy source, `CGO_ENABLED=1 go build -o bin/dc-management-tool ./cmd/server`
- **Runtime stage**: `debian:bookworm-slim`
  - Install `ca-certificates`, `sqlite3`, `curl`
  - Copy binary, migrations/, templates/, static/ from builder
  - Copy `docker-entrypoint.sh`
  - Expose 8080

### 2. `docker-entrypoint.sh`
- Set env defaults (DATABASE_PATH, SERVER_ADDRESS, etc.)
- Start app in background (migrations run automatically)
- Poll `/health` until ready (max 30s)
- Apply `migrations/seed_data.sql` via `sqlite3` CLI (with `|| true` for idempotency)
- `wait $APP_PID` to keep container alive

### 3. `docker-compose.yml`
- Define a template service `app` with build context
- Override with 5 named services: `app-1` through `app-5`
- Ports: 8081-8085 mapped to container 8080
- Each gets its own named volume for `/app/data`
- Health checks on each

### 4. `scripts/run-parallel-tests.sh`
- Orchestration script that:
  1. Accepts args: `--suites "phase-4,phase-5,phase-6"` or `--all`
  2. Builds and starts N containers (one per suite)
  3. Waits for all health checks
  4. Launches N parallel background processes, each running playwright-cli commands from its assigned test plan against its assigned port
  5. Waits for all to finish, collects exit codes
  6. Tears down containers
  7. Reports pass/fail summary

### 5. `scripts/run-suite.sh`
- Runs a single test plan against a given port
- Args: `--plan testing-plans/phase-4-products-master.md --port 8081`
- Uses `playwright-cli` commands to execute the test steps
- This is what each parallel agent/process calls

### 6. Makefile additions
```makefile
test-docker-build:    ## Build the Docker image
test-docker-up:       ## Start 5 isolated app containers (ports 8081-8085)
test-docker-down:     ## Stop and remove all test containers
test-docker-parallel: ## Run all test plans in parallel across Docker containers
test-docker-suite:    ## Run a single suite: make test-docker-suite PLAN=phase-4 PORT=8081
```

## Key Design Decisions

1. **playwright-cli on host, app in Docker**: Playwright-cli already works on the host. Only the app needs isolation. No need to put playwright inside Docker.

2. **5 containers by default**: Enough for 5 parallel agents. Can scale up by adding more services to docker-compose.

3. **Each container is fully independent**: Fresh DB, fresh migrations, fresh seed data. No shared state.

4. **Test plans stay as-is**: The existing `.md` test plans in `testing-plans/` don't need modification. Agents just need to know which port to target instead of hardcoded `localhost:8080`.

5. **Seed data applied inside container**: The entrypoint runs `sqlite3 $DB < seed_data.sql` after migrations complete. Each container starts with identical, known state.

## Files to Modify

- `Makefile` — append Docker test targets
- `.gitignore` — add `tests/results/` if not already present

## Verification

1. `make test-docker-build` — image builds successfully
2. `make test-docker-up` — 5 containers start, all healthy
3. `curl http://localhost:8081/health` through `8085` — all return OK
4. Manual test: `playwright-cli open http://localhost:8081/login` — app loads
5. `make test-docker-down` — clean teardown
6. Parallel test: run playwright-cli against two different ports simultaneously — no state conflicts
