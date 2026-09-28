# Vacuum dependency speed test — 2026-09-28

The dependency-only upgrade uses libopenapi `v0.41.1`, libopenapi-validator `v0.14.1`, and `github.com/pb33f/go-yaml v0.1.0`. Compatible Doctor and libasyncapi YAML migrations are local. A separate Vacuum change removes repeated document scans from the OWASP array and string limit rules.

## Dependency-only result

Median end-to-end `vacuum lint` seconds, before → upgraded. Lower is faster. This table excludes the OWASP fix.

| Spec | Regular | Turbo | Hard | Hard + turbo |
| --- | ---: | ---: | ---: | ---: |
| petstore.yaml | 0.045 → 0.045 | 0.040 → 0.039 | 0.062 → 0.049 | 0.041 → 0.040 |
| mistral.yaml | 0.140 → 0.124 | 0.089 → 0.086 | 0.158 → 0.153 | 0.155 → 0.152 |
| neon.yaml | 0.160 → 0.125 | 0.120 → 0.085 | 0.240 → 0.241 | 0.267 → 0.236 |
| ld.yaml | 0.395 → 0.347 | 0.320 → 0.324 | 2.286 → 2.423 | 2.198 → 2.313 |
| plaid.yml | 1.083 → 0.751 | 0.477 → 0.440 | 5.504 → 3.930 | 3.778 → 3.729 |
| stripe.yaml | 0.832 → 0.782 | 0.811 → 0.707 | 27.578 → 24.125 | 26.101 → 25.996 |

## After the separate OWASP fix

The rule fix changes hard-mode work; regular and turbo use the dependency-only numbers above. Median seconds for the upgraded build → upgraded build with the fix:

| Spec | Hard | Hard + turbo |
| --- | ---: | ---: |
| petstore.yaml | 0.049 → 0.079 | 0.040 → 0.040 |
| mistral.yaml | 0.153 → 0.106 | 0.152 → 0.097 |
| neon.yaml | 0.241 → 0.178 | 0.236 → 0.213 |
| ld.yaml | 2.423 → 0.632 | 2.313 → 0.529 |
| plaid.yml | 3.930 → 1.820 | 3.729 → 1.566 |
| stripe.yaml | 24.125 → 3.042 | 25.996 → 3.078 |

The default five-second rule limit dropped `owasp-array-limit` and `owasp-string-limit` on Stripe before the fix. With a 60-second rule limit, the baseline Stripe hard-mode median was 27.578 s; the dependency-only median was 24.125 s. The fixed binary completed in 3.042 s (3.042–3.567 s across measured runs). At the ordinary five-second limit, the fixed binary completed every spec and mode with no skipped rules.

Selected median peak resident memory (MiB):

| Spec / mode | Before | Dependencies | With rule fix |
| --- | ---: | ---: | ---: |
| ld.yaml / regular | 744 | 575 | 582 |
| plaid.yml / regular | 710 | 632 | 622 |
| stripe.yaml / hard | 1357 | 1235 | 1204 |

## Method and validation

- Apple M4 Max, 16 logical CPUs, macOS, Go 1.26.4. Six unchanged files from `../demo/speed-test`; their SHA-256 hashes and sizes are in [the results](results/libopenapi-0.41.1.json).
- Each of the 24 spec/mode combinations had one warm-up and three measured runs per binary. Runs were serial; case order was shuffled with a fixed seed and binary order rotated. Numbers are wall-clock CLI time and peak RSS from macOS `/usr/bin/time -l`. Small differences, especially below 0.1 s, are near run-to-run noise.
- All runs used `lint --silent --no-style --no-banner --no-update-check --timeout 60 --lookup-timeout 10000` with an empty config, isolated XDG config, and no `VACUUM_*` environment variables. Turbo and hard flags were added for their respective modes. The larger limits let the original Stripe hard-mode rules finish; they did not disable any check. The ordinary limits were checked again on the final binary.
- All 288 interleaved runs exited with the expected lint status (0 for clean or 1 for violations). None logged a rule or lookup timeout. The three complete Spectral-report matrices each produced 216,626 findings across the 24 cases; the final binary produced the same count under ordinary limits. IDs, paths, severity, range, and message content match. Three Neon `oas3-schema` findings vary only in the order of two error clauses, and repeated runs show that both orders already occur within every binary.
- `GOWORK=off go test ./...` passed for the isolated libasyncapi migration. Doctor and Vacuum full suites passed with temporary module files pointing to the two local migrations. The final Vacuum full suite passed after the OWASP fix. All three binaries built successfully in the corresponding module graph.

## Local release boundary

Vacuum still names published Doctor `v0.0.80` and libasyncapi `v0.0.2` because their YAML-compatible replacements have not been released. The local Doctor migration is commit `ccf0e90`; libasyncapi is `e79f259`. Both are on local `codex/vacuum-yaml-migration` branches. Vacuum commits are `9e6935c0` (dependencies) and `6d4a82ac` (rule performance) on `codex/dependency-benchmarks`. The upgraded Vacuum build and test evidence uses temporary `-modfile` replacements; a clean `GOWORK=off` build without those replacements needs compatible upstream releases and updated pins. No upstream release or PR was published.

The YAML fork changes the exported `*yaml.Node` package identity. Go library users and custom plugin authors will need to update their YAML imports when consuming the eventual Vacuum release.
