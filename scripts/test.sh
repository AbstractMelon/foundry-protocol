#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

echo "==> go build"
go build ./...

echo "==> go vet"
go vet ./...

echo "==> go test"
go test ./... -count=1

echo "==> all checks passed"