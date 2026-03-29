#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UI_DIR="${ROOT_DIR}/html-report/ui"
COREPACK_HOME="${COREPACK_HOME:-${ROOT_DIR}/.cache/corepack}"
YARN_CACHE_FOLDER="${YARN_CACHE_FOLDER:-${ROOT_DIR}/.cache/yarn}"
YARN_CMD=()

if ! command -v node >/dev/null 2>&1; then
  echo "node is required to build vacuum from source." >&2
  exit 1
fi

if command -v yarn >/dev/null 2>&1; then
  YARN_CMD=(yarn)
elif command -v corepack >/dev/null 2>&1; then
  YARN_CMD=(corepack yarn)
else
  echo "yarn or corepack is required to build the HTML report UI assets." >&2
  exit 1
fi

mkdir -p "${COREPACK_HOME}" "${YARN_CACHE_FOLDER}"

export COREPACK_HOME
export YARN_CACHE_FOLDER

cd "${UI_DIR}"
"${YARN_CMD[@]}" install --frozen-lockfile --prefer-offline
"${YARN_CMD[@]}" build

if [ "${KEEP_NODE_MODULES:-0}" != "1" ]; then
  rm -rf node_modules
fi
