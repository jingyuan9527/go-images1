#!/usr/bin/env bash
set -euo pipefail

APP_NAME="webapp"
PORT="${PORT:-8808}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

if ! command -v docker &>/dev/null; then
  error "Docker is not installed."
  echo "Install Docker first:"
  echo "  curl -fsSL https://get.docker.com | sh"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

if [ ! -f Dockerfile ]; then
  error "Cannot find Dockerfile. Make sure you're in the project root."
  exit 1
fi

info "Building Docker image (this may take a few minutes)..."
docker build -t "$APP_NAME" .

if docker ps -a --format '{{.Names}}' | grep -q "^${APP_NAME}$"; then
  warn "Removing existing container with the same name..."
  docker rm -f "$APP_NAME" >/dev/null
fi

PROXY_ARGS=""
if [ -n "${HTTP_PROXY:-}" ] || [ -n "${http_proxy:-}" ]; then
  PROXY="${HTTP_PROXY:-${http_proxy}}"
  PROXY_ARGS="$PROXY_ARGS -e HTTP_PROXY=$PROXY -e HTTPS_PROXY=$PROXY"
  info "Using proxy: $PROXY"
fi

info "Starting container on port $PORT..."
docker run -d \
  -p "$PORT:8808" \
  $PROXY_ARGS \
  --name "$APP_NAME" \
  --restart unless-stopped \
  "$APP_NAME" >/dev/null

sleep 2

IP=$(curl -s --max-time 3 http://localhost:"$PORT/" 2>/dev/null | head -1 || true)
if [ -n "$IP" ]; then
  info "Deployment successful!"

  HOST_IP=$(ip -4 addr show | grep -oP 'inet \K[\d.]+' | grep -v '127.0.0.1\|^172\.' | head -1 || echo "localhost")
  echo ""
  echo "  Local:   http://localhost:$PORT"
  echo "  Network: http://$HOST_IP:$PORT"
  echo ""
else
  warn "Container started but service is not responding yet."
  echo "Check logs: docker logs $APP_NAME"
fi
