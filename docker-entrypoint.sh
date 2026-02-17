#!/bin/bash
set -e

export DATABASE_PATH="${DATABASE_PATH:-/app/data/dc_management.db}"
export SERVER_ADDRESS="${SERVER_ADDRESS:-:8080}"
export SESSION_SECRET="${SESSION_SECRET:-docker-test-secret}"
export APP_ENV="${APP_ENV:-development}"
export UPLOAD_PATH="${UPLOAD_PATH:-/app/static/uploads}"

# Start app in background (migrations auto-run in main.go)
./dc-management-tool &
APP_PID=$!

# Wait for health endpoint
echo "Waiting for app to be ready..."
for i in $(seq 1 60); do
    # Extract port from SERVER_ADDRESS (e.g., ":8081" -> "8081")
    APP_PORT="${SERVER_ADDRESS#:}"
    if curl -sf "http://localhost:${APP_PORT}/health" > /dev/null 2>&1; then
        echo "App is ready!"
        break
    fi
    if [ "$i" -eq 60 ]; then
        echo "App failed to start within 60s"
        exit 1
    fi
    sleep 1
done

# Seed data (idempotent — duplicate key errors suppressed)
if [ -f /app/migrations/seed_data.sql ]; then
    echo "Seeding database..."
    sqlite3 "$DATABASE_PATH" < /app/migrations/seed_data.sql || true
    echo "Seed complete."
fi

wait $APP_PID
