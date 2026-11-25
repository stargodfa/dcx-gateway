#!/usr/bin/env bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${REPO_ROOT}/dist/dcx-gateway-arm64"
OUTPUT="${DIST_DIR}/dcx-gateway"

if [ ! -d "${REPO_ROOT}/cmd/dcx-gateway" ]; then
  echo "error: gateway entry ./cmd/dcx-gateway not found" >&2
  exit 1
fi

mkdir -p "${DIST_DIR}"

echo "==> Building linux/arm64 gateway"
echo "    output: ${OUTPUT}"
(
  cd "${REPO_ROOT}"
  GOOS=linux GOARCH=arm64 go build -o "${OUTPUT}" ./cmd/dcx-gateway
)

ls -lh "${OUTPUT}"
echo "==> Build complete"
