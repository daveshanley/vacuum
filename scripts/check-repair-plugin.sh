#!/bin/sh
# Exercise a compiled Go repair plugin through single-file and multi-file lint.
set -eu
binary=${1:?usage: check-repair-plugin.sh /path/to/vacuum}
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT HUP INT TERM
mkdir "$scratch/plugins"
(cd "$repo" && GOWORK=off go build -buildmode=plugin -o "$scratch/plugins/repair.so" ./cmd/test_data/repair_plugin)
cat > "$scratch/rules.yaml" <<'RULES'
rules:
  title:
    given: $.info.title
    severity: error
    autoFixFunction: clearTitle
    then: {function: falsy}
RULES
for file in one two three; do
 printf 'openapi: 3.0.3\ninfo: {title: Bad, version: "1"}\npaths: {}\n' > "$scratch/$file.yaml"
done
"$binary" lint --no-update-check --silent --fix --functions "$scratch/plugins" --ruleset "$scratch/rules.yaml" "$scratch/one.yaml"
"$binary" lint --no-update-check --silent --fix --functions "$scratch/plugins" --ruleset "$scratch/rules.yaml" "$scratch/two.yaml" "$scratch/three.yaml"
for file in one two three; do
 if grep -q Bad "$scratch/$file.yaml"; then
  printf 'Repair was not saved: %s\n' "$file" >&2
  exit 1
 fi
 "$binary" lint --no-update-check --silent --ruleset "$scratch/rules.yaml" "$scratch/$file.yaml"
done
printf '%s\n' 'Go repair plugin verified through both lint paths.'
