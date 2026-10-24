# Ruleset compatibility

Run the pinned OpenLint core-function cases with Go:

```sh
GOWORK=off go test ./compatibility -count=1 -v
```

No Node, Python, downloads or external CLI is needed. These tests also run with `go test ./...`.

The corpus contains all 31 cases in `v1/conformance/functions` from
[openlint/spec at a3f1272](https://github.com/openlint/spec/tree/a3f1272397aeef51a1f736cc95b7d45be6ff1179/v1/conformance/functions).
The fixtures are unchanged. Their Apache-2.0 license is in `testdata/openlint/LICENSE`.
This is a working draft, not a released standard.

Each case runs through Vacuum's ruleset loader, function preflight, engine and
Spectral report model. The test compares the fields named by the upstream case,
including the exact number of findings. Invalid options and execution errors
cannot count as expected findings.

The log separates three results:

- `COMPATIBLE`: matches the upstream expectation.
- `DIFFERENT`: matches the documented Vacuum behavior in `differences.json`.
- `UNSUPPORTED`: rejects the function or options with the expected preflight error.

A changed difference fails the test. A difference that becomes compatible also
fails, so its entry can be removed. A green Go test run does not mean every case
is compatible.

This corpus covers core functions only. It does not certify the full ruleset
specification, source ranges, diagnostic wording, Node APIs or third-party npm
packages. When updating it, pin the new revision, copy the whole function corpus,
check the case count and review each difference. Do not edit upstream expectations
to match Vacuum.

JavaScript import checks run separately:

```sh
GOWORK=off go test ./motor ./plugin/jsruleset -run 'TestJavaScript|TestPrepare' -count=1
```

Both `@stoplight/spectral-*` and `@openlint/openlint-*` names expose the same
supported `core`, `functions` and `formats` helpers. The function bridge includes
`or`. Other exports remain unsupported.
