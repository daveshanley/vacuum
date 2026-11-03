# Autofix benchmark measurements

Measured on 2026-11-02, macOS arm64, Go 1.26.10. Production code: `f2c9a12a`; benchmark instrumentation is committed with this report. Three samples per phase; values below are medians. Inputs include the six existing `demo/speed-test` specs and three synthetic fixtures. No source files were written. The benchmark uses the recommended OpenAPI ruleset, no plugin callbacks, and no remote lookup. These are local measurements, not a cross-machine guarantee.

| Input | Bytes | Lint ms | Plan ms | Apply ms | Repeat ms | Verified fixes |
|---|---:|---:|---:|---:|---:|---:|
| ld.yaml | 1,927,409 | 363.84 | 455.38 | 452.16 | 455.58 | 0 |
| mistral.yaml | 292,380 | 80.48 | 102.06 | 234.49 | 100.56 | 2 |
| neon.yaml | 369,544 | 102.08 | 117.89 | 121.81 | 120.87 | 0 |
| petstore.yaml | 21,302 | 13.04 | 14.46 | 14.10 | 12.89 | 0 |
| plaid.yml | 2,870,618 | 813.08 | 959.66 | 957.38 | 959.96 | 0 |
| stripe.yaml | 6,051,965 | 890.59 | 1,220.11 | 1,232.52 | 1,206.77 | 0 |
| synthetic-repair | 145 | 0.65 | 0.55 | 1.08 | 0.55 | 1 |
| synthetic-1000-repairs | 41,990 | 494.53 | 516.04 | 562.54 | 33.89 | 1000 |
| synthetic-shared-alias | 190 | 0.63 | 0.70 | 0.65 | 0.65 | 0 |

`Plan` includes baseline lint. `Apply` includes planning and, when edits exist, reparsing and verification. Inputs with no eligible edits measure planning/refusal, not a second lint pass. Every sample reported zero execution errors; every repeat pass reported zero further repairs. Repair counts come from verified fixed results, not from a lower violation total.

| Input | Baseline findings | Final findings | Candidate bytes | Fixes by rule |
|---|---:|---:|---:|---|
| ld.yaml | 6073 | 6073 | 1,927,409 |  |
| mistral.yaml | 1066 | 1064 | 379,669 | `oas-schema-check`: 2 |
| neon.yaml | 1281 | 1281 | 369,544 |  |
| petstore.yaml | 94 | 94 | 21,302 |  |
| plaid.yml | 8895 | 8895 | 2,870,618 |  |
| stripe.yaml | 21839 | 21839 | 6,051,965 |  |
| synthetic-repair | 5 | 4 | 253 | `duplicated-entry-in-enum`: 1 |
| synthetic-1000-repairs | 3002 | 2002 | 93,036 | `duplicated-entry-in-enum`: 1000 |
| synthetic-shared-alias | 8 | 8 | 190 |  |

| Input | Lint allocated MiB | Plan allocated MiB | Apply allocated MiB | Repeat allocated MiB |
|---|---:|---:|---:|---:|
| ld.yaml | 640.42 | 722.38 | 722.88 | 722.08 |
| mistral.yaml | 99.73 | 117.17 | 260.52 | 117.10 |
| neon.yaml | 131.85 | 149.58 | 149.66 | 149.62 |
| petstore.yaml | 11.80 | 13.00 | 12.95 | 13.02 |
| plaid.yml | 849.35 | 985.30 | 982.48 | 984.85 |
| stripe.yaml | 941.31 | 1,227.14 | 1,228.21 | 1,227.31 |
| synthetic-repair | 0.98 | 1.03 | 2.12 | 1.02 |
| synthetic-1000-repairs | 848.33 | 871.06 | 964.91 | 49.40 |
| synthetic-shared-alias | 0.84 | 0.88 | 0.88 | 0.88 |

Allocation figures are cumulative bytes allocated during each call (`runtime.MemStats.TotalAlloc`), **not peak RSS or retained heap**. A full GC precedes each sample and is excluded from its time. The benchmark ran without another local test/build workload. The raw samples include allocation counts in [benchmark-results.jsonl](benchmark-results.jsonl).

The complete compiled benchmark process used **1,137,098,752 bytes maximum resident set size (1,084.42 MiB)**. macOS separately reported **1,111,918,104 bytes peak memory footprint (1,060.41 MiB)**. These are whole-process maxima across all inputs and phases, not per-call memory estimates. `/usr/bin/time -l` measured the binary, excluding compilation. Total wall time was 39.09 seconds.

The independent de-slop review measured an earlier planning bottleneck on 8,000 small enum repairs: 1.804 seconds before indexing, 35.17 milliseconds after adding prefix-conflict and pointer indexes (about 51 times faster in that isolated probe). Final planning also sorts findings so worker completion order cannot change a plan. The corpus measurements above cover the final implementation.

Reproduce:

```sh
GOWORK=off GOTOOLCHAIN=go1.26.10 go build -o /tmp/vacuum-autofix-bench ./design/autofix/bench
/usr/bin/time -l /tmp/vacuum-autofix-bench -corpus /path/to/demo/speed-test -runs 3
```
