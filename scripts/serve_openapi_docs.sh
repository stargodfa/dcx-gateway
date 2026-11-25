#!/usr/bin/env bash
set -eu

DOCS_PORT="${DOCS_PORT:-8090}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVE_DIR="${REPO_ROOT}/docs/api"

if [ ! -f "${SERVE_DIR}/openapi.yaml" ]; then
  echo "error: ${SERVE_DIR}/openapi.yaml not found" >&2
  exit 1
fi

if [ ! -f "${SERVE_DIR}/swagger-ui/index.html" ]; then
  echo "error: ${SERVE_DIR}/swagger-ui/index.html not found" >&2
  exit 1
fi

HOST_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
if [ -z "${HOST_IP}" ]; then
  HOST_IP="127.0.0.1"
fi

echo "Serving OpenAPI spec: http://${HOST_IP}:${DOCS_PORT}/openapi.yaml"
echo "Serving Swagger UI:   http://${HOST_IP}:${DOCS_PORT}/swagger-ui/"
cd "${SERVE_DIR}"
python3 -m http.server "${DOCS_PORT}"
