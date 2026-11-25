#!/usr/bin/env bash
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 deploy@<host>" >&2
  exit 1
fi

TARGET="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REMOTE_ROOT="/opt/dcx-gateway-docs"

if [ ! -f "${REPO_ROOT}/docs/api/openapi.yaml" ]; then
  echo "error: docs/api/openapi.yaml not found" >&2
  exit 1
fi

if [ ! -d "${REPO_ROOT}/docs/api/swagger-ui" ]; then
  echo "error: docs/api/swagger-ui not found" >&2
  exit 1
fi

for asset in swagger-ui.css swagger-ui-bundle.js swagger-ui-standalone-preset.js; do
  if [ ! -f "${REPO_ROOT}/docs/api/swagger-ui/${asset}" ]; then
    echo "error: docs/api/swagger-ui/${asset} not found" >&2
    exit 1
  fi
done

echo "==> Preparing remote docs directory ${REMOTE_ROOT}"
ssh "${TARGET}" "mkdir -p '${REMOTE_ROOT}/docs/api' '${REMOTE_ROOT}/scripts' '${REMOTE_ROOT}/systemd'"

echo "==> Uploading OpenAPI docs"
scp "${REPO_ROOT}/docs/api/openapi.yaml" "${TARGET}:${REMOTE_ROOT}/docs/api/openapi.yaml"
scp -r "${REPO_ROOT}/docs/api/swagger-ui" "${TARGET}:${REMOTE_ROOT}/docs/api/"
scp "${REPO_ROOT}/scripts/serve_openapi_docs.sh" "${TARGET}:${REMOTE_ROOT}/scripts/serve_openapi_docs.sh"
scp "${REPO_ROOT}/systemd/dcx-gateway-interface.service" "${TARGET}:${REMOTE_ROOT}/systemd/dcx-gateway-interface.service"

ssh "${TARGET}" "chmod +x '${REMOTE_ROOT}/scripts/serve_openapi_docs.sh'"

cat <<NEXT

Next steps on ${TARGET}:
  sudo cp ${REMOTE_ROOT}/systemd/dcx-gateway-interface.service /etc/systemd/system/
  sudo systemctl daemon-reload
  sudo systemctl enable dcx-gateway-interface
  sudo systemctl restart dcx-gateway-interface
  journalctl -u dcx-gateway-interface -f

OpenAPI spec: http://<ip>:8090/openapi.yaml
Swagger UI:   http://<ip>:8090/swagger-ui/

NEXT
