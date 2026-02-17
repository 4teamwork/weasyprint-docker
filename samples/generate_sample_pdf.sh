#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${ROOT_DIR}/out"
mkdir -p "${OUT_DIR}"

NEW_URL="${NEW_URL:-http://localhost:8080/}"
NEW_BEARER_TOKEN="${NEW_BEARER_TOKEN:-}"

if [[ -z "${NEW_BEARER_TOKEN}" ]]; then
  echo "NEW_BEARER_TOKEN is not set. The service requires JWT auth."
  echo "Set NEW_BEARER_TOKEN and rerun."
  exit 1
fi

echo "Generating PDF from: ${NEW_URL}"
curl -sS -f \
  -X POST "${NEW_URL}" \
  -H "Authorization: Bearer ${NEW_BEARER_TOKEN}" \
  -F "html=@${ROOT_DIR}/sample.html;type=text/html" \
  -F "css=@${ROOT_DIR}/sample.css;type=text/css" \
  -F "asset.logo=@${ROOT_DIR}/logo.svg;type=image/svg+xml;filename=logo.svg" \
  -o "${OUT_DIR}/new.pdf"

echo "Done. Output file: ${OUT_DIR}/new.pdf"
