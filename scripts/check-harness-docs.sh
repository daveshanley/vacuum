#!/bin/sh
# Verify published harness contracts and examples against a built vacuum binary.
set -eu
binary=${1:?usage: check-harness-docs.sh /path/to/vacuum /path/to/quobix.com}
docs=${2:?usage: check-harness-docs.sh /path/to/vacuum /path/to/quobix.com}
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT HUP INT TERM
for contract in suite result; do
    "$binary" test --schema "$contract" > "$scratch/$contract.json"
    cmp "$scratch/$contract.json" "$docs/static/vacuum/schemas/test-$contract-v1.json"
done
for fixture in suite.yaml rules.yaml valid.yaml missing-id.yaml; do
    cmp "$repo/harness/testdata/agent/$fixture" "$docs/static/vacuum/examples/agent-testing/$fixture"
done
"$binary" test "$docs/static/vacuum/examples/agent-testing/suite.yaml" > "$scratch/result.json"
printf '%s\n' 'Harness documentation schemas and examples verified.'
