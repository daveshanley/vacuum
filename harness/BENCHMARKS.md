# Harness suite benchmarks

Measured on 2026-10-20, darwin/arm64, Apple M4 Max, Go 1.26. The baseline is
`691e36b7` with `benchmark_test.go`; the comparison adds the compiled ruleset
schema cache. Values below are medians of three single-iteration samples.
They establish a local baseline, not a performance guarantee.

Each case uses one rule and an OpenAPI document containing 50 small component
schemas (5,525 bytes). The reference workload adds eight local reference files
and has a 5,821-byte entry document. JavaScript runs one custom function.
Autofix performs four complete engine invocations per case and checks semantic
output, post-fix findings, and idempotence. Every measured suite must pass.

The measurement includes suite loading, ruleset validation, document preparation,
rule execution, and construction of the detached report. It excludes CLI process
startup and JSON output encoding. Fixtures are local temporary files; filesystem
caches may be warm. Each call creates fresh user rules and documents.

| Workload, 100 cases | Before, ms | After, ms | Before, allocated MB | After, allocated MB |
| --- | ---: | ---: | ---: | ---: |
| lint | 361.42 | 240.82 | 496.44 | 362.58 |
| references | 458.38 | 335.14 | 573.86 | 439.84 |
| javascript | 381.25 | 250.96 | 502.26 | 368.29 |
| autofix | 1556.22 | 1135.83 | 2071.67 | 1536.06 |

Allocated bytes are cumulative allocation churn, not peak or retained heap.
The allocation profile attributed about 26% of baseline allocated bytes to
`rulesets.CreateRuleSetUsingJSON`, including repeated compilation of the same
ruleset schema. The change retains one compiled validator keyed by schema text;
it does not cache user rules, documents, findings, or reference graphs. Changing
the exported schema text replaces that one cache entry. Concurrent validation
and independent user rulesets have race-tested regression coverage.

Document/model construction remains a substantial cost. Reusing mutable document
state between cases would need a separate ownership design and is not part of
this optimization. No retained-heap reduction is claimed.

## Reproduce

```sh
GOWORK=off GOCACHE=/tmp/vacuum-research-go-cache go test ./harness \
  -run '^$' -bench '^BenchmarkSuite$' -benchmem -benchtime=1x -count=3

GOWORK=off GOCACHE=/tmp/vacuum-research-go-cache go test ./harness \
  -run '^$' -bench '^BenchmarkSuite/autofix/cases100$' -benchtime=2x \
  -cpuprofile=/tmp/harness-cpu.pprof -memprofile=/tmp/harness-memory.pprof \
  -o /tmp/harness-profile.test

go tool pprof -top -alloc_space /tmp/harness-profile.test /tmp/harness-memory.pprof
```

Run the timing comparison without concurrent test or profiling jobs. Use longer
runs and more samples before setting a release performance budget. The benchmark
also includes 10-case variants to check how cost grows with suite size.
