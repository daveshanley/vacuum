# Verified autofix implementation

The 157-rule matrix is the scope record. It classifies repair opportunities; it
is not a claim that every diagnostic can be changed without owner input.

## Surfaces

- `repair.BuiltinPlan` produces guarded proposals and explicit declines. The
  provider fixtures cover the 15 automatic-candidate rule IDs in the matrix.
- `motor.PlanFixes` / `PlanFixesWithPolicy` perform baseline lint and planning.
- `motor.ValidateFixPlan` validates a proposal independently. By default the
  proposal must exactly equal the current built-in/policy proposal. A caller
  can approve an external plan by supplying the SHA-256 of its JSON encoding
  in `RepairApproval.ApprovedSHA256`. Approval is separate from plan contents.
  It approves intent, not a proof of API contract equivalence. Plans without
  original diagnostic targets never increase the verified repair count.
- `motor.RunRepairs` coordinates built-ins, supplied policy, and enabled trusted
  plugin callbacks. `RunPluginRepairs` selects only callbacks. The returned
  candidate and final findings are available without filesystem writes. The
  returned plan describes built-in/policy edits; legacy callback effects are
  represented by the candidate bytes and verified result records.
- `lint --fix-plan NEW_FILE` exports proposals and dispositions. It is a preview;
  it does not assert that candidate verification succeeded. Patch operations
  describe the proposed changes. `--fix --fix-builtins` applies eligible edits.
  `--fix-file OUTPUT` keeps the original file. Multi-input runs cannot share one
  plan or repaired output file. Report stdout remains in its requested format.
- Both `schema` and `schema lint` use the same repair coordinator. Schema stdin
  and bundle mode cannot overwrite inputs.
- LSP code actions return a versioned `documentChanges` edit. They never write
  files or mutate the open buffer. Changed/closed documents and changed config
  discard pending proposals. Root-mode multi-file LSP repairs are declined.

`RuleFunctionResult.DiagnosticCode` is additive report metadata. `RepairNode`
identifies the authored owner independently of the displayed range and is not
serialized. Plan target `code` identifies the guarded repair case, or an actual
producer diagnostic code. External targets without a producer/provider code
must use `reviewed-edit`; arbitrary invented codes are rejected.

## Local policy

Policy values are exact owner decisions, not generated placeholders. The first
providers support contact, license, descriptions, summaries and operation IDs.
The other policy candidates in the audit remain reviewed edits; accepting a
rule ID is not evidence of a working provider.

```yaml
version: 1
policies:
  - id: team-contact
    rule: info-contact
    diagnostic: contact-missing
    scope:
      source: ./api.yaml
      pointer: /info
    value:
      name: API Team
      email: api@example.com
      url: https://example.com/support
  - id: operation-docs
    rule: operation-description
    diagnostic: description-missing
    scope:
      source: ./api.yaml
      pointer: /paths/~1pets/get
    catalog: ./operation-descriptions.yaml
```

Catalog:

```yaml
api.yaml:
  listPets: List the pets visible to the authenticated caller.
```

Invoke with `--fix --fix-builtins --fix-policy policy.yaml`. Enable optional
rules explicitly in the ruleset or with `--hard-mode`; `info-contact`, for
example, is not in the recommended OpenAPI ruleset.

Sources and catalog names resolve relative to the policy file. Catalog lookup
requires a unique operation ID and exact source identity. Missing entries,
conflicting existing values, wrong scopes and overlapping edits are declined.
Unknown fields, duplicate assignments, wildcard scopes, unsupported rule/code
pairs and missing catalog snapshots fail validation. Policy/catalog bytes are
hashed and retained privately; a submitted external plan cannot grant itself
policy authority. Current policies insert missing fields; they do not replace
an existing empty/null or conflicting value.

## Source and verification contracts

JSON Patch operations use sequential RFC 6902 ordering. Removing the document
root is rejected because it cannot produce a document for lint verification. The envelope adds
source hashes, expected absence, policy provenance, dependency hashes and the
specific YAML key-tag / duplicate-key edits used by providers. Duplicate-key
removals run in their own batch. Array-changing groups own the containing array
so another provider cannot apply stale positional pointers.

One built-in/policy batch performs baseline lint and candidate lint. Trusted
plugin callbacks need an additional execution pass, then the same final
verification. A callback must mutate in place, return the supplied node or nil,
and leave its original diagnostic resolved. No-op callbacks do not count.

Verification preserves rules, severities, ignore files, reference options and
timeout/cancellation controls. It requires complete lint, checks original
targets, rejects new diagnostic identities and rejects changed inline ignore
directives. It does not claim that relinting proves arbitrary semantic intent.
Existing line-bearing description-duplication messages use stable codes during
comparison, so reformatting cannot manufacture a new diagnostic.

OpenAPI external reference bytes are captured from the baseline Rolodex.
Different captured versions of one dependency are rejected. Verification uses
an in-memory filesystem with no network/filesystem fallback. External edits
are declined. AsyncAPI/Arazzo external-source verification is currently declined
until those loaders supply an equivalent frozen collection.

Source streams containing multiple YAML documents are rejected before rewrite.
Automatic proposals decline anchored/aliased trees, ambiguous duplicate keys,
unsupported scalar values and edits that would lose comments. Removing or
reordering array entries in documents containing references is conservative:
those cases require review. Tag sorting is limited to 1,000 entries. Source
parsing is limited to 64 MiB and depth 256; patch application limits operations
to 10,000 and cumulative node-copy cost to 256 MiB. These are refusal limits,
not a hard process RSS guarantee.

CLI writing checks original bytes and source identity, rejects symlink/non-file
inputs and hardlink aliases, preserves permissions, stages in the destination
directory, and renames. Hash-check plus rename is not filesystem compare-and-swap
against an uncooperative writer. Versioned LSP edits are the editor boundary.

## Validation

Focused tests cover sequential patch behavior, exact numbers, rollback,
policy/catalog scope, source changes, ignore files, unchanged diagnostics across
formatting, multi-document refusal, callback verification, frozen HTTP inputs,
JSON Schema CLI repair, AsyncAPI providers, and versioned LSP buffer edits.
The whole repository test suite passed during implementation. Final checks and
measured benchmark results belong in the PR and BENCHMARKS.md.

The original `probe` remains an engine-level regression tool: no-op and returned
replacement callbacks no longer count as fixes; inserted JSON fields serialize
as valid JSON. It tests the lower-level callback API, not the final coordinator.

Reproduce planning/repair measurements without modifying the corpus:

```sh
GOWORK=off go run ./design/autofix/bench -corpus /path/to/demo/speed-test -runs 3
GOWORK=off go test ./repair ./motor ./cmd ./language-server -run 'Repair|Patch|Policy|AutoFix' -count=1
GOWORK=off go test -race ./repair ./motor ./language-server -run 'Repair|Patch|AutoFixTransactionOutcome' -count=1
GOWORK=off go run ./design/autofix/check
```

No WASM experience, website, Go version or dependency versions change here.
Overlay export, general policy inheritance/globs, external multi-file writes,
and ambiguous schema transformations remain outside this implementation.
