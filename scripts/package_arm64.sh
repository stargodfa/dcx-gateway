#!/usr/bin/env bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PACKAGE_DIR="${REPO_ROOT}/dist/dcx-gateway-arm64"
ARCHIVE="${REPO_ROOT}/dist/dcx-gateway-arm64.tar.gz"
BINARY="${PACKAGE_DIR}/dcx-gateway"

if [ ! -f "${BINARY}" ]; then
  echo "==> Binary not found, running build_arm64.sh"
  bash "${SCRIPT_DIR}/build_arm64.sh"
fi

for asset in swagger-ui.css swagger-ui-bundle.js swagger-ui-standalone-preset.js; do
  if [ ! -f "${REPO_ROOT}/docs/api/swagger-ui/${asset}" ]; then
    echo "error: docs/api/swagger-ui/${asset} not found" >&2
    exit 1
  fi
done

echo "==> Preparing package directory"
mkdir -p "${PACKAGE_DIR}/configs" "${PACKAGE_DIR}/docs" "${PACKAGE_DIR}/systemd"

cp "${REPO_ROOT}/configs/config.example.yaml" "${PACKAGE_DIR}/configs/config.example.yaml"
cp "${REPO_ROOT}/readme.md" "${PACKAGE_DIR}/readme.md"
rm -f "${PACKAGE_DIR}/configs/config.yaml" "${PACKAGE_DIR}/configs/config.dev.yaml"

rm -rf "${PACKAGE_DIR}/migrations" "${PACKAGE_DIR}/docs"
cp -R "${REPO_ROOT}/migrations" "${PACKAGE_DIR}/migrations"
cp -R "${REPO_ROOT}/docs" "${PACKAGE_DIR}/docs"
cp "${REPO_ROOT}/systemd/dcx-gateway.service" "${PACKAGE_DIR}/systemd/dcx-gateway.service"

echo "==> Creating archive"
rm -f "${ARCHIVE}"
(
  cd "${REPO_ROOT}/dist"
  tar -czf "$(basename "${ARCHIVE}")" dcx-gateway-arm64
)

echo "==> Package: ${ARCHIVE}"
echo "==> Package summary:"
tar -tzf "${ARCHIVE}" | head -n 40
