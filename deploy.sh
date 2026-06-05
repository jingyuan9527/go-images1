#!/usr/bin/env bash
set -euo pipefail

APP_NAME="go-images"
PORT="${PORT:-8808}"
BIND_ADDR="${BIND_ADDR:-127.0.0.1}"

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

info "Building Docker image..."
docker build -t "$APP_NAME" .

if docker ps -a --format '{{.Names}}' | grep -q "^${APP_NAME}$"; then
  warn "Stopping and removing existing container..."
  docker stop "$APP_NAME" >/dev/null 2>&1 || true
  docker rm "$APP_NAME" >/dev/null 2>&1 || true
fi

PROXY_ARGS=""
proxy="${HTTP_PROXY:-${http_proxy:-}}"
if [ -n "$proxy" ]; then
  PROXY_ARGS="-e HTTP_PROXY=$proxy -e HTTPS_PROXY=$proxy"
  info "Using proxy: $proxy"
fi

AUTH_ARGS=""
if [ -n "${ACCESS_PASSWORD:-}" ]; then
  AUTH_ARGS="-e ACCESS_PASSWORD=$ACCESS_PASSWORD"
  info "Access password enabled"
fi

info "Starting container on ${BIND_ADDR}:${PORT}..."
docker run -d \
  -p "${BIND_ADDR}:${PORT}:8808" \
  $PROXY_ARGS \
  $AUTH_ARGS \
  --name "$APP_NAME" \
  --restart unless-stopped \
  "$APP_NAME" >/dev/null

sleep 2

if curl -s -o /dev/null -w "" --max-time 3 "http://localhost:${PORT}/" 2>/dev/null; then
  info "Deployment successful!"
  echo ""
  echo "  Local:   http://localhost:${PORT}"
  echo "  Network: http://$(ip -4 addr show | grep -oP 'inet \K[\d.]+' | grep -v '127.0.0.1\|^172\.' | head -1):${PORT}"
  echo ""
  echo "  To change bind address, run:"
  echo "    BIND_ADDR=0.0.0.0 ./deploy.sh"
  echo "    BIND_ADDR=127.0.0.1 ./deploy.sh"
else
  warn "Container started but service is not responding yet."
  echo "Check logs: docker logs $APP_NAME"
fi