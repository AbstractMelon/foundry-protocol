#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

LAUNCH_CLIENT=0
NO_GATEWAY=0
for arg in "$@"; do
  case "$arg" in
    --client) LAUNCH_CLIENT=1 ;;
    --no-gateway) NO_GATEWAY=1 ;;
    *)
      echo "Unknown option: $arg" >&2
      echo "Usage: ./scripts/dev.sh [--client] [--no-gateway]" >&2
      exit 2
      ;;
  esac
done

pids=()
cleanup() {
  echo ""
  echo "Stopping dev environment..."
  for pid in "${pids[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

if ! command -v air >/dev/null 2>&1; then
  echo "[dev] air not found, installing..."
  go install github.com/air-verse/air@latest
fi

echo "==> Foundry Protocol dev environment (air hot-reload)"
echo "  World server : ws://localhost:8090/ws   (restarts on .go and content/*.yaml changes)"
echo "  Gateway      : http://localhost:8080"

air -c scripts/air.world.toml &
pids+=($!)

if [ "$NO_GATEWAY" = "0" ]; then
  air -c scripts/air.gateway.toml &
  pids+=($!)
fi

if [ "$LAUNCH_CLIENT" = "1" ]; then
  sleep 3
  if command -v godot >/dev/null 2>&1; then
    echo "[dev] launching client in dev mode..."
    godot --path client -- --dev &
    pids+=($!)
  else
    echo "[dev] godot not found on PATH — open client/ in the Godot editor instead."
  fi
fi

cat <<'EOF'

Cheatsheet
  Play game      : godot --path client -- --dev                 (dev: skip menu, auto-connect to local world)
  Normal join    : godot --path client                          (menu -> name -> gateway -> servers -> join)
                  set a name at launch:  godot --path client -- --user Zed
  Open editor    : godot -e --path client
  Run tests      : ./scripts/test.sh
  In-game (dev)  : /give copper 500  /set iron 10  /clear  /help  (in chat box)
  Edit content   : content/*.yaml — the world server hot-reloads on save
  Add textures   : drop a PNG into assets/ and reference it in the yaml via "texture:"

Press Ctrl+C to stop everything.
EOF

wait