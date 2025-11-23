#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${TARGET_DIR:-${1:-.}}"
SKIP_BUILD="${SKIP_BUILD:-0}"
PAYLOAD_FILE=""
PAYLOAD_DIR=""
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_BASE_URL="${SCRIPT_BASE_URL:-https://raw.githubusercontent.com/Kavis1/3X-UI-testapi/main/api-hardener}"
PAYLOAD_URL="${PAYLOAD_URL:-${SCRIPT_BASE_URL}/payload.b64}"

cleanup() {
  if [[ -n "${PAYLOAD_FILE}" && -f "${PAYLOAD_FILE}" ]]; then
    rm -f "${PAYLOAD_FILE}"
  fi
  [[ -n "${TMPDIR:-}" && -d "${TMPDIR}" ]] && rm -rf "${TMPDIR}"
}
trap cleanup EXIT

TMPDIR="$(mktemp -d)"
PAYLOAD_FILE="${TMPDIR}/payload.tgz"
PAYLOAD_DIR="${TMPDIR}/payload"

if [[ -f "${SCRIPT_DIR}/payload.b64" ]]; then
  cp "${SCRIPT_DIR}/payload.b64" "${TMPDIR}/payload.b64"
else
  echo ">> Downloading payload from ${PAYLOAD_URL}"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${PAYLOAD_URL}" -o "${TMPDIR}/payload.b64"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "${TMPDIR}/payload.b64" "${PAYLOAD_URL}"
  else
    echo "curl or wget required to download payload" >&2
    exit 1
  fi
fi

if command -v base64 >/dev/null 2>&1; then
  base64 -d "${TMPDIR}/payload.b64" > "${PAYLOAD_FILE}"
else
  cat > "${TMPDIR}/decode_payload.py" <<'PY'
import base64
import pathlib
import sys

src = pathlib.Path(sys.argv[1])
dst = pathlib.Path(sys.argv[2])
dst.write_bytes(base64.b64decode(src.read_bytes()))
PY
  python "${TMPDIR}/decode_payload.py" "${TMPDIR}/payload.b64" "${PAYLOAD_FILE}"
fi

mkdir -p "${PAYLOAD_DIR}"
tar -xzf "${PAYLOAD_FILE}" -C "${PAYLOAD_DIR}"

echo ">> Copying hardened API payload into ${TARGET_DIR}"
if command -v rsync >/dev/null 2>&1; then
  rsync -a "${PAYLOAD_DIR}/"/ "${TARGET_DIR}/"
else
  cp -R "${PAYLOAD_DIR}/." "${TARGET_DIR}/"
fi

if [[ "${SKIP_BUILD}" != "1" ]]; then
  echo ">> Running go mod tidy..."
  (cd "${TARGET_DIR}" && go mod tidy)
  echo ">> Building api-guard CLI..."
  (cd "${TARGET_DIR}" && go build -o api-guard ./cmd/api-guard)
fi

cat <<'DONE'
API hardener applied.
Next:
  1) Запустите: ./api-guard install
  2) Перезапустите панель.
  3) В панели появится вкладка “API” для управления токенами и лимитами.
DONE
