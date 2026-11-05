#!/usr/bin/env bash
# Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
# SPDX-License-Identifier: MIT
#
# Builds the vacuum CLI as a WebAssembly module, to run in a browser.
#
# Usage: browser/build.sh <output directory>
#
# Writes vacuum.<hash>.wasm.gz, wasm_exec.<hash>.js and manifest.json. The module
# is gzipped for its loader to inflate (DecompressionStream), so any static host
# can serve it without Content-Encoding headers. SKIP_COMPRESS=1 writes a plain
# vacuum.<hash>.wasm instead (faster local rebuilds). Set WASM_OPT to a Binaryen
# wasm-opt binary to shrink the module further. Like release builds, it includes
# the HTML report UI, building those assets with npm when they are missing.
#
# bubbletea has no js/wasm platform hooks yet (charmbracelet/bubbletea#1790).
# Until it does, this script builds against a temporary copy of the exact
# bubbletea version in go.mod with the stubs in browser/patches/bubbletea added.
# go.mod is never changed, and native builds never see the copy.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ $# != 1 ]]; then
  echo 'Usage: browser/build.sh <output directory>' >&2
  exit 2
fi
mkdir -p "$1"
out="$(cd "$1" && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
export GOWORK=off

bubbletea_dir="$(go list -m -f '{{.Dir}}' charm.land/bubbletea/v2)"
bubbletea_version="$(go list -m -f '{{.Version}}' charm.land/bubbletea/v2)"
if [[ -f "$bubbletea_dir/tty_js.go" ]]; then
  echo "bubbletea $bubbletea_version supports js/wasm; building without the patch."
else
  echo "Patching bubbletea $bubbletea_version for js/wasm in a temporary copy."
  cp -R "$bubbletea_dir" "$work/bubbletea"
  chmod -R u+w "$work/bubbletea"
  for patch in browser/patches/bubbletea/*.go.tmpl; do
    cp "$patch" "$work/bubbletea/$(basename "$patch" .tmpl)"
  done
  cat > "$work/go.work" <<EOF
go $(go list -m -f '{{.GoVersion}}')

use $PWD

replace charm.land/bubbletea/v2 $bubbletea_version => $work/bubbletea
EOF
  export GOWORK="$work/go.work"
fi

# html-report embeds the same UI bundle the release binaries ship.
if [[ ! -f html-report/ui/build/static/js/vacuumReport.js ]]; then
  ./scripts/build-ui-assets.sh
fi

version="$(git describe --tags --always 2>/dev/null || echo dev)"
commit="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
GOOS=js GOARCH=wasm go build -trimpath -tags html_report_ui \
  -ldflags "-s -w -X main.version=$version -X main.commit=$commit -X main.date=$(date -u +%Y-%m-%d)" \
  -o "$work/vacuum.wasm" .

if [[ -n "${WASM_OPT:-}" ]]; then
  "$WASM_OPT" "$work/vacuum.wasm" -Oz --enable-bulk-memory --enable-sign-ext \
    --enable-nontrapping-float-to-int -o "$work/vacuum.opt.wasm"
  mv "$work/vacuum.opt.wasm" "$work/vacuum.wasm"
fi

hash="$(shasum -a 256 "$work/vacuum.wasm" | cut -c1-12)"
runtime="wasm_exec.$hash.js"
if [[ "${SKIP_COMPRESS:-0}" == 1 ]]; then
  module="vacuum.$hash.wasm"
  cp "$work/vacuum.wasm" "$out/$module"
else
  module="vacuum.$hash.wasm.gz"
  gzip -9 -c "$work/vacuum.wasm" > "$out/$module"
fi
# The loader must come from the same toolchain that compiled the module.
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/$runtime"
chmod 644 "$out/$module" "$out/$runtime"

# Swap the manifest in only once every new file exists, then drop the previous
# build, so a page loading mid-build never points at a missing module. `bytes`
# is the decompressed size, which the page's progress bar counts against.
cat > "$out/manifest.json.tmp" <<EOF
{
  "module": "$module",
  "runtime": "$runtime",
  "bytes": $(wc -c < "$work/vacuum.wasm" | tr -d ' '),
  "version": "$version",
  "commit": "$commit",
  "go": "$(go env GOVERSION)"
}
EOF
mv "$out/manifest.json.tmp" "$out/manifest.json"
for stale in "$out"/vacuum.*.wasm* "$out"/wasm_exec.*.js; do
  [[ -e "$stale" && "$stale" != *".$hash."* ]] && rm -f "$stale"
done
echo "Wrote $out/$module ($(du -h "$out/$module" | cut -f1)) and manifest.json"
