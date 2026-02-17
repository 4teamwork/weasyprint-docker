#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${ROOT_DIR}/out"
mkdir -p "${OUT_DIR}"

OLD_URL="${OLD_URL:-http://localhost:18080/}"
NEW_URL="${NEW_URL:-http://localhost:8080/}"
NEW_BEARER_TOKEN="${NEW_BEARER_TOKEN:-}"

echo "Generating OLD pdf from: ${OLD_URL}"
curl -sS -f \
  -X POST "${OLD_URL}" \
  -F "html=@${ROOT_DIR}/sample.html;type=text/html" \
  -F "css=@${ROOT_DIR}/sample.css;type=text/css" \
  -F "asset.logo=@${ROOT_DIR}/logo.svg;type=image/svg+xml;filename=logo.svg" \
  -o "${OUT_DIR}/old.pdf"

echo "Generating NEW pdf from: ${NEW_URL}"
if [[ -z "${NEW_BEARER_TOKEN}" ]]; then
  echo "NEW_BEARER_TOKEN is not set. The new service requires JWT auth."
  echo "Set NEW_BEARER_TOKEN and rerun."
  exit 1
fi

curl -sS -f \
  -X POST "${NEW_URL}" \
  -H "Authorization: Bearer ${NEW_BEARER_TOKEN}" \
  -F "html=@${ROOT_DIR}/sample.html;type=text/html" \
  -F "css=@${ROOT_DIR}/sample.css;type=text/css" \
  -F "asset.logo=@${ROOT_DIR}/logo.svg;type=image/svg+xml;filename=logo.svg" \
  -o "${OUT_DIR}/new.pdf"

echo
echo "Binary SHA256 checksums:"
sha256sum "${OUT_DIR}/old.pdf" "${OUT_DIR}/new.pdf"

if command -v pdftoppm >/dev/null 2>&1; then
  rm -f "${OUT_DIR}"/old-*.png "${OUT_DIR}"/new-*.png
  pdftoppm -png "${OUT_DIR}/old.pdf" "${OUT_DIR}/old"
  pdftoppm -png "${OUT_DIR}/new.pdf" "${OUT_DIR}/new"

  if cmp -s "${OUT_DIR}/old-1.png" "${OUT_DIR}/new-1.png"; then
    echo "Page 1 raster output is identical."
  else
    echo "Page 1 raster output differs. Inspect files in ${OUT_DIR}."
  fi
else
  echo "pdftoppm not found. Install poppler-utils for visual raster comparison."
fi

echo "Done. Output files are in ${OUT_DIR}"
