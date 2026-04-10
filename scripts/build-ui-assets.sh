#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UI_DIR="${ROOT_DIR}/html-report/ui"
NPM_CACHE_DIR="${NPM_CACHE_DIR:-${ROOT_DIR}/.cache/npm}"

if ! command -v node >/dev/null 2>&1; then
  echo "node is required to build vacuum from source." >&2
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required to build the HTML report UI assets." >&2
  exit 1
fi

mkdir -p "${NPM_CACHE_DIR}"

export npm_config_cache="${NPM_CACHE_DIR}"

cd "${UI_DIR}"
npm ci --prefer-offline
npm run build

if [ "${KEEP_NODE_MODULES:-0}" != "1" ]; then
  rm -rf node_modules
fi
