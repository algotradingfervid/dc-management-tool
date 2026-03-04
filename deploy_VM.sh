#!/usr/bin/env bash
set -euo pipefail

REMOTE_USER="naren"
REMOTE_HOST="100.121.122.93"   # Tailscale IP
REMOTE_DIR="/opt/dc-management"
BINARY_NAME="dc-management-tool"
LOCAL_BINARY="bin/${BINARY_NAME}-linux"

echo "==> Building Linux binary..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -o "$LOCAL_BINARY" ./cmd/server
echo "    Built: $(du -sh "$LOCAL_BINARY" | cut -f1) — $(file "$LOCAL_BINARY" | grep -o 'ELF.*stripped')"

echo "==> Copying binary to ${REMOTE_HOST}..."
rsync -avz --progress "$LOCAL_BINARY" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}/${BINARY_NAME}.new"

# nginx: client_max_body_size 100M is set in /etc/nginx/nginx.conf (one-time manual setup)

echo "==> Swapping binary and restarting service..."
ssh "${REMOTE_USER}@${REMOTE_HOST}" bash << 'ENDSSH'
  set -euo pipefail
  mv /opt/dc-management/dc-management-tool.new /opt/dc-management/dc-management-tool
  chmod +x /opt/dc-management/dc-management-tool
  sudo systemctl restart dc-management
  sleep 2
  sudo systemctl is-active dc-management
ENDSSH

echo "==> Done. App is running at http://${REMOTE_HOST}:8080"
