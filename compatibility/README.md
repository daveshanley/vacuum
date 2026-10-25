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

All 31 cases run. 23 match upstream. Eight deliberate differences preserve
Vacuum behavior and are checked against exact results in `differences.json`.
A changed result fails the test. These differences are not compatibility passes.

This corpus covers core functions only. It does not certify the full ruleset
specification, source ranges, diagnostic wording, Node APIs or third-party npm
packages. When updating it, pin the new revision, copy the whole function corpus,
check the case count and review each result. Do not edit upstream expectations
to match Vacuum.

JavaScript import checks run separately:

```sh
GOWORK=off go test ./motor ./plugin/jsruleset -run 'TestJavaScript|TestPrepare' -count=1
```

Both `@stoplight/spectral-*` and `@openlint/openlint-*` names expose the same
supported `core`, `functions` and `formats` helpers. The function bridge includes
`or` and `unreferencedReusableObject`. Other exports remain unsupported.

## Function behavior

- `schema` reports all failures and retains validation of newer schema keywords
  by default. Set `allErrors: false` only when one failure per value is wanted.
- `dialect` accepts `draft4`, `draft6`, `draft7`, `draft2019-09` and `draft2020-12`.
  An explicit `$schema` in the schema takes precedence.
- Missing schema fields fail by default. Set `forceValidation: false` for an
  optional field.
- `alphabetical` keeps lexical string ordering and reports each array inversion.
- `truthy` rejects empty collections. `pattern` still checks non-string values,
  `casing` rejects empty strings, and `length` keeps UTF-8 byte limits.
- `xor` still rejects inputs with neither required property, including scalars.
- `unreferencedReusableObject` checks the existing reference index. Set
  `reusableObjectsLocation` to a local JSON Pointer, such as `#/defs`, and select
  that container with `given`. Use `resolved: false` to retain reference nodes.
