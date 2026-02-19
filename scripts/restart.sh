#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_DIR/bin/dc-management-tool"

cd "$PROJECT_DIR"

echo "==> Stopping running server..."
pkill -f 'dc-management-tool' 2>/dev/null && echo "    Stopped dc-management-tool" || echo "    No running process found"
sleep 1

echo "==> Generating templ files..."
"$HOME/go/bin/templ" generate ./components

echo "==> Building binary..."
go build -o "$BINARY" ./cmd/server
echo "    Built: $BINARY"

echo "==> Loading environment..."
if [ -f "$PROJECT_DIR/.env" ]; then
  set -a
  # shellcheck source=/dev/null
  source "$PROJECT_DIR/.env"
  set +a
  echo "    Loaded .env"
else
  echo "    No .env found, using defaults"
  export APP_ENV="${APP_ENV:-development}"
  export SERVER_ADDRESS="${SERVER_ADDRESS:-:8080}"
  export DATABASE_PATH="${DATABASE_PATH:-./data/dc_management.db}"
  export SESSION_SECRET="${SESSION_SECRET:-dev-secret-change-in-production}"
  export UPLOAD_PATH="${UPLOAD_PATH:-./static/uploads}"
fi

echo "==> Starting server (${SERVER_ADDRESS:-:8080})..."
exec "$BINARY"
