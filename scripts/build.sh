#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BIN_DIR="bin"
mkdir -p "$BIN_DIR"

echo "==> Building foundry protocol"

echo "  gateway -> $BIN_DIR/gateway"
go build -trimpath -ldflags "-s -w" -o "$BIN_DIR/gateway" ./cmd/gateway

echo "  world   -> $BIN_DIR/world"
go build -trimpath -ldflags "-s -w" -o "$BIN_DIR/world" ./cmd/world

ls -lh "$BIN_DIR"
echo "==> Build complete"