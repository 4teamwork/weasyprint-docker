#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT_DIR}/.env"

cd "${ROOT_DIR}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Missing ${ENV_FILE}."
  echo "Create it from template: cp ${ROOT_DIR}/.env.example ${ENV_FILE}"
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

if [[ -z "${AUTH_AUTHORITY:-}" || -z "${AUTH_AUDIENCE:-}" ]]; then
  echo "AUTH_AUTHORITY and AUTH_AUDIENCE must be set in ${ENV_FILE}."
  exit 1
fi

if ! command -v bwrap >/dev/null 2>&1; then
  echo "bwrap not found on PATH. Install bubblewrap (e.g. sudo apt-get install -y bubblewrap)."
  exit 1
fi

if ! command -v weasyprint >/dev/null 2>&1; then
  echo "weasyprint not found on PATH. Install weasyprint (e.g. pip install weasyprint)."
  exit 1
fi

exec go run ./cmd/pdfservice/main.go
