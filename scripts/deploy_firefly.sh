#!/usr/bin/env bash
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 deploy@<host>" >&2
  exit 1
fi

TARGET="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ARCHIVE="${REPO_ROOT}/dist/dcx-gateway-arm64.tar.gz"
REMOTE_BASE="/opt/dcx-gateway"
REMOTE_ARCHIVE="/tmp/dcx-gateway-arm64.tar.gz"
REMOTE_APP_DIR="${REMOTE_BASE}/dcx-gateway-arm64"

if [ ! -f "${ARCHIVE}" ]; then
  echo "==> Package not found, running package_arm64.sh"
  bash "${SCRIPT_DIR}/package_arm64.sh"
fi

echo "==> Uploading ${ARCHIVE} to ${TARGET}:${REMOTE_ARCHIVE}"
scp "${ARCHIVE}" "${TARGET}:${REMOTE_ARCHIVE}"

echo "==> Extracting on remote host"
ssh "${TARGET}" "mkdir -p '${REMOTE_BASE}' && tar -xzf '${REMOTE_ARCHIVE}' -C '${REMOTE_BASE}'"

echo "==> Checking remote config"
ssh "${TARGET}" "if [ ! -f '${REMOTE_APP_DIR}/configs/config.dev.yaml' ]; then echo 'config.dev.yaml not found. Create it from the example before starting:'; echo '  cd ${REMOTE_APP_DIR}'; echo '  cp configs/config.example.yaml configs/config.dev.yaml'; echo '  vi configs/config.dev.yaml'; fi"

cat <<NEXT

Next steps on ${TARGET}:
  cd ${REMOTE_APP_DIR}
  cp configs/config.example.yaml configs/config.dev.yaml   # only if config.dev.yaml does not exist
  vi configs/config.dev.yaml
  sudo cp systemd/dcx-gateway.service /etc/systemd/system/
  sudo systemctl daemon-reload
  sudo systemctl enable dcx-gateway
  sudo systemctl restart dcx-gateway
  journalctl -u dcx-gateway -f

NEXT
