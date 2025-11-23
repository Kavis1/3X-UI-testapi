#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${TARGET_DIR:-${1:-.}}"
SKIP_BUILD="${SKIP_BUILD:-0}"
PAYLOAD_FILE=""
PAYLOAD_DIR=""
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_BASE_URL="${SCRIPT_BASE_URL:-https://raw.githubusercontent.com/Kavis1/3X-UI-testapi/main/api-hardener}"
PAYLOAD_URL="${PAYLOAD_URL:-${SCRIPT_BASE_URL}/payload.b64}"
GO_VERSION="${GO_VERSION:-1.22.5}"
GO_ARCHIVE="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="${GO_URL:-https://go.dev/dl/${GO_ARCHIVE}}"
GO_BIN=""

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

ensure_go() {
  if command -v go >/dev/null 2>&1; then
    GO_BIN="$(command -v go)"
    return
  fi
  echo ">> Go не найден, скачиваю ${GO_URL}"
  local archive="${TMPDIR}/${GO_ARCHIVE}"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${GO_URL}" -o "${archive}"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "${archive}" "${GO_URL}"
  else
    echo "curl или wget необходимы для загрузки Go" >&2
    exit 1
  fi
  tar -xzf "${archive}" -C "${TMPDIR}"
  GO_BIN="${TMPDIR}/go/bin/go"
  if [[ ! -x "${GO_BIN}" ]]; then
    echo "Не удалось распаковать Go" >&2
    exit 1
  fi
  export PATH="${TMPDIR}/go/bin:${PATH}"
}

echo ">> Copying hardened API payload into ${TARGET_DIR}"
if command -v rsync >/dev/null 2>&1; then
  rsync -a "${PAYLOAD_DIR}/" "${TARGET_DIR}/"
else
  cp -R "${PAYLOAD_DIR}/." "${TARGET_DIR}/"
fi

# Fallback: ensure binary exists even if copy was skipped by older shells
if [[ ! -f "${TARGET_DIR}/api-guard.linux-amd64" && -f "${PAYLOAD_FILE}" ]]; then
  tar -xzf "${PAYLOAD_FILE}" -C "${TARGET_DIR}" api-guard.linux-amd64 || true
fi

if [[ -f "${TARGET_DIR}/api-guard.linux-amd64" ]]; then
  mv -f "${TARGET_DIR}/api-guard.linux-amd64" "${TARGET_DIR}/api-guard"
fi

if [[ -f "${TARGET_DIR}/api-guard" ]]; then
  chmod +x "${TARGET_DIR}/api-guard"
  if [[ -w "/usr/local/bin" ]]; then
    ln -sf "${TARGET_DIR}/api-guard" /usr/local/bin/api-guard
  fi
fi

if [[ "${SKIP_BUILD}" != "1" ]]; then
  if [[ ! -f "${TARGET_DIR}/go.mod" ]]; then
    echo ">> go.mod не найден в ${TARGET_DIR}, сборка пропущена (использую встроенный бинарник api-guard)"
  else
    ensure_go
    echo ">> Running go mod tidy..."
    (cd "${TARGET_DIR}" && "${GO_BIN}" mod tidy)
    echo ">> Building api-guard CLI..."
    (cd "${TARGET_DIR}" && "${GO_BIN}" build -tags toolsignore -o api-guard ./cmd/api-guard)
  fi
fi

API_BIN="${TARGET_DIR}/api-guard"
if [[ ! -x "${API_BIN}" ]] && command -v api-guard >/dev/null 2>&1; then
  API_BIN="$(command -v api-guard)"
fi

if [[ -x "${API_BIN}" ]]; then
  echo ">> Running api-guard install..."
  "${API_BIN}" install
fi

cat <<'DONE'
API hardener applied.
Next:
  1) Перезапустите панель.
  2) В панели появится вкладка "API" для управления токенами и лимитами.
DONE
