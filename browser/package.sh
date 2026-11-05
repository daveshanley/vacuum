#!/usr/bin/env bash
# Packages the browser build of vacuum as a release asset.
#
# Usage: browser/package.sh
#
# Builds the module with browser/build.sh and writes
# browser/dist/vacuum-online-engine.tar (manifest.json, the gzipped module and
# its wasm_exec loader). GoReleaser attaches it to each release.
set -euo pipefail
cd "$(dirname "$0")/.."

rm -rf browser/dist
browser/build.sh browser/dist/engine
(cd browser/dist/engine && tar -cf ../vacuum-online-engine.tar manifest.json vacuum.*.wasm* wasm_exec.*.js)
echo "Wrote browser/dist/vacuum-online-engine.tar"
