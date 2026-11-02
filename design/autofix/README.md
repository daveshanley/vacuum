# Built-in autofix: audit and implementation plan

Audit date: 2026-11-02. Source: `6dab7db05afd81b2a6fdb6cd41175abcecabe516` (production v0.36.2). Scope: all exported built-in OpenAPI, OWASP, AsyncAPI, JSON Schema and Arazzo rules, their registered functions, and the current repair execution/output paths. This document records the reviewed plan. The implemented surfaces, limits and verification are tracked in [IMPLEMENTATION.md](IMPLEMENTATION.md).

## Decision

Build autofix as a shared, verifiable repair service. Start with narrow repairs whose outcome follows from the document and the specification. Add policy-driven repairs next. Expose contract-changing repairs as reviewable plans.

Detection does not generally determine a repair. A missing contact email can have many correct values. An `allOf` conflict can be resolved by changing either branch, with different contracts. A missing security declaration does not identify the authentication system. Deleting a constraint until lint passes is not a correct fix.

The useful target is **a repair disposition for every finding**, with a reason when automatic repair is unavailable. Do not target 100% automatic mutation or equate fewer findings with correctness.

The deterministic set is small. The policy class covers contact, license, descriptions, IDs and responses; prioritizing these is a product hypothesis, not measured user demand. Language-model agents already draft many of those values. Vacuum's distinct contribution is the gate around any proposal: plan against a snapshot, apply privately, reparse, relint, prove the targeted diagnostic is gone and nothing new appeared, then write atomically. **Make verification of externally proposed plans a first-class entry point**, not only a step behind built-in providers. Built-in providers, policy providers, LSP actions and agents then share one acceptance path.

Order the work by value and independence:

1. Engine hardening ships first as a standalone bug fix for existing plugin users. It does not depend on diagnostic codes or the plan API.
2. Design the plan contract and external-plan verification together. Include a policy identity/hash in provenance. Settle the small local policy format when implementing the first policy providers; a complete policy language is not a prerequisite for automatic repairs.
3. Built-in automatic providers follow, then policy providers.

The [plan-crush review](PLAN-CRUSH.md) records the rejected mechanisms, evidence and remaining implementation gates.

## Inventory and coverage

The exported registries contain **157 unique rule IDs using 96 distinct function names**. None of these rule definitions sets `AutoFixFunction`. The inventory includes optional rules, not only the default recommended ruleset. Several functions are shared core functions (`truthy`, `pattern`, `schema`, etc.); the rules do not each have a separate custom function.

| Family | A: automatic candidate | P: policy required | R: review required | I: inactive detector | O: operational | Total |
|---|---:|---:|---:|---:|---:|---:|
| OpenAPI | 10 | 25 | 33 | 1 | 0 | 69 |
| OWASP | 0 | 19 | 7 | 0 | 0 | 26 |
| AsyncAPI | 2 | 17 | 14 | 1 | 0 | 34 |
| JSON Schema | 2 | 1 | 7 | 0 | 0 | 10 |
| Arazzo | 1 | 5 | 11 | 0 | 1 | 18 |
| **Total** | **15** | **67** | **72** | **2** | **1** | **157** |

Classification is a design assessment, not measured fix coverage:

- **A**: At least one narrow diagnostic case has a deterministic repair candidate. The provider must prove all guards. Other findings under the same rule can still require policy or review.
- **P**: An explicit, scoped policy can supply the missing intent. Automatic application is possible only after the owner has selected the policy and its values. A policy must not silently authorize changes to unrelated contracts.
- **R**: A reviewable edit or migration is possible, but selecting it requires a decision. Some may later yield more narrow A cases. This is not a statement that code cannot assist.
- **I**: The function currently returns no findings for this built-in configuration. Do not count registration as repair coverage.
- **O**: Validation did not complete. Recover the required inputs or execution conditions; do not manufacture a document edit.

[The full matrix](MATRIX.md) gives **every rule** its function source, proposed repair, guard/refusal conditions, and focused fixture requirements. [inventory.json](inventory.json) retains each original rule's selectors, formats, actions, options and metadata with its assessment. [The coverage checker](check/main.go) compares this snapshot with the live exported registries and fails on missing, extra or changed rules.

The Arazzo adapter delegates to libopenapi-validator v0.16.2. Its diagnostic groups were also traced in `arazzo/semantics.go`, `sources_targets.go`, `expressions.go`, `inputs.go` and `advice.go` in that dependency. Exact repeated Arazzo parameter declarations provide another narrow automatic case; other parameter errors do not.

The classifications describe the least restrictive useful repair route identified for each rule. They are not mutually exclusive capabilities of a whole rule. For example, `path-params` is A for `required: false`, but missing parameter definitions need policy and overlapping routes need review.

## What already exists

The premise is partly correct: no built-in repair implementations are registered. The infrastructure is more than an empty hook:

| Component | Current behavior | Work needed |
|---|---|---|
| [model/autofix.go](../../model/autofix.go), [model/rules.go](../../model/rules.go) | Named callback takes target node, document and rule context; returns node/error. Rule carries `AutoFixFunction`. | Add an explicit plan/outcome contract without breaking Go plugin callbacks. |
| [plugin/plugin_manager.go](../../plugin/plugin_manager.go) | Go plugins can register named repair callbacks. | Keep one registration owner; select built-in planners by rule/diagnostic and retain named plugin callbacks. Reject name collisions; no silent override. |
| [motor/auto_fix_transaction.go](../../motor/auto_fix_transaction.go) | Clone graph per fixing rule; serialize fix transactions; preserve node identity on publication; stage repeated edits; rollback callback errors; guard cancellation/timeouts. | Reuse graph cloning and rollback protections under one coordinator. Separate analysis from mutation; verify actual effects and conflicts. |
| [motor/rule_applicator.go](../../motor/rule_applicator.go), [motor/document_applicator.go](../../motor/document_applicator.go) | Apply callbacks while executing rules. | Collect proposals from a stable snapshot and apply after lint workers finish. |
| [motor/autofix_output.go](../../motor/autofix_output.go) | Serializes repaired root with YAML marshal; reports marshal errors. | Preserve JSON output as JSON; inspect the existing renderBundledOutput conversion used by apply-overlay before adding another serializer. Keep reusable serialization below cmd; preserve YAML presentation where possible. |
| [cmd/lint_cmd.go](../../cmd/lint_cmd.go), [cmd/lint_process_multifile.go](../../cmd/lint_process_multifile.go) | Plugin fixes wired into CLI; fixed details/counts; output file; temp backup; shared output rejected for multiple inputs. | Register built-ins centrally; add preview; atomic source-checked writes; publish only verified results. |
| [cmd/schema.go](../../cmd/schema.go) | JSON Schema lint commands have no fix flags. | Add the same supported repair flow to both `schema` and `schema lint`. |
| [language-server/server.go](../../language-server/server.go) | `quickfix` opens diagnostic documentation. | Return versioned WorkspaceEdit actions from the same planner. |
| [rulesets/rule_fixes.go](../../rulesets/rule_fixes.go) | Human guidance. | Keep guidance distinct from an executable repair and its prerequisites. |

`ModifiedSpec` represents one root document. It is not a multi-document change set. A referenced source outside that root needs an explicit source edit and persistence plan. Do not mark an external finding fixed because an in-memory referenced node changed.

### Confirmed behavior probes

[probe/main.go](probe/main.go) runs three small library-level cases against the audited engine. Observed output:

```text
noop:               fixed=1; description is still empty
return-replacement: fixed=1; returned replacement is ignored; description is still empty
insert-json-field:  fixed=1; json.Valid(ModifiedSpec)=false
```

The third case starts with valid JSON and inserts a plain YAML scalar key/value. YAML marshal produces `..., x-audit: value}`. Existing JSON node styles can preserve valid JSON for some edits, but the current writer does not guarantee it for new nodes.

The callback return value is discarded in `applyAutoFixesToResults`. `commitAutoFixes` marks queued findings fixed without checking that a change occurred or rerunning the detector. These are concrete reasons to harden the engine before installing built-in fixers. Keep the probes as audit evidence; replace their observations with regression expectations when implementing the repair service.

These defects affect plugin users of `--fix` today. Fix them independently of the repair service: detect unchanged document transactions and report them as not fixed, define the return-value contract, and keep JSON input as JSON output. A callback can legitimately edit a parent or sibling while leaving its target unchanged. A changed transaction proves an edit occurred; only a fresh validation pass can prove the finding was resolved. None of this requires new public types.

The current CLI uses `os.WriteFile` after making a temp backup. It does not use an atomic same-directory replacement or verify that the source still matches the linted bytes. File replacement needs both checks.

Concurrency and external-source publication require focused tests during implementation. This audit identifies design risks; it does not claim a reproduced data race or completed multi-file corruption test.

## First automatic repair set

The following 15 rule IDs contain the proposed initial automatic cases. This list is deliberately narrower than each rule's complete diagnostic set.

| Rule IDs | Eligible edit |
|---|---|
| `duplicated-entry-in-enum`, `json-schema-enum-values-compatible` | Remove typed, structurally equal enum duplicates. |
| `openapi-tags-alphabetical`, `asyncapi-3-tags-alphabetical` | Stable name sort for well-formed tag objects. |
| `openapi-tags`, `asyncapi-3-tags-uniqueness` | Remove fully identical tag objects; retain conflicting same-name objects for review. |
| `oas2-api-schemes` | Remove repeated scheme strings. |
| `duplicate-paths`, `operation-parameters`, `arazzo-parameter` | Remove exact duplicate complete definitions in one scope. |
| `path-params` | Set `required: true` on an existing path parameter; remove exact duplicate parameter definitions. |
| `operation-success-response` | Quote a numeric response code without changing its value or response. |
| `no-unnecessary-combinator` | Unwrap a singleton only when the wrapper is a proven identity and reference/scope guards pass. |
| `oas-schema-check` | Remove redundant supported `enum` beside a compatible `const`; remove self-dependencies. |
| `json-schema-dependent-required-defined` | Remove redundant self-dependency entries. |

Implement duplicates, response-key quoting and required-path flags before schema rewrites. Singleton unwrapping is the highest-risk item in this set and should ship later, after scope/annotation tests. Sorting is a selected presentation policy: enable it only when the ordering rule is enabled.

Global guards also apply to these cases: source must be writable and owned; incoming pointers to array positions must be known or the edit declined; anchored values and alias reachability must remain valid; comments must be retained or the edit offered for review. Exact duplicate content is not permission to discard a separately named anchor or useful comment.

## Findings that change the design

1. **Repair a diagnostic, not a rule name.** `oas-schema-check`, `path-params`, schema validators and Arazzo groups each emit different kinds of findings. Add stable diagnostic codes and structured payloads at their producers. Do not match English `Message` strings. Allow additive optional fields in reports so existing consumers keep working.
2. **Use the canonical source location.** `StartNode` can be a key, owner, resolved target or synthetic node. Store document URI, JSON Pointer tokens, expected node kind/value and source version/hash. Keep display `Path` and `Paths` unchanged in meaning. A shared component is edited once even if it produces many use-site paths.
3. **Preserve validation completeness.** `arazzo-validation-incomplete` is an execution/input state. `asyncapi-3-document-unresolved` returns no function findings. `oas3-no-$ref-siblings` is an unconditional no-op. `json-schema-ref-valid` is different: its function is a no-op but the engine emits real reference findings before rule execution.
4. **Do not implement a universal `truthy` or `pattern` fixer.** The same function can request a contact email, an operation description or an address constraint. The repair needs the rule/action/diagnostic identity and domain policy.

Per-rule refusals are recorded in [the matrix](MATRIX.md). The ones that most constrain provider design:

- Do not infer `type: object` from `properties`; adding a type narrows the accepted set.
- Do not deduplicate `oneOf` branches; removing an identical branch can turn an invalid instance valid.
- A security declaration is not proof that TLS, rate limiting, authentication or JWT validation is implemented.
- OAS 3.0 Reference Objects, OAS 3.1 schemas, Path Items and JSON Schema dynamic references treat `$ref` siblings differently. Never apply a global remove-siblings operation.
- Do not prune potentially unused public components by default. External consumers, discriminator mappings, security names and cycles need wider reachability checks.

Additional engine diagnostics such as `resolving-references`, `schema-build-failure`, `build-index`, `index-failure` and `circular-references` are outside the 157 exported rule IDs. Include them in the disposition API with source/environment evidence. Do not break valid recursive schemas to remove a circular-reference finding. Parse errors and unsupported dialects can prevent enough analysis to propose any edit.

## Proposed shared API

Names below are design sketches, not committed public interfaces:

```text
PlanFixes(snapshot, findings, policy, selection) -> FixPlan
ValidateFixPlan(snapshot, plan) -> verified in-memory changes + final lint result
WriteFixPlan(verified changes, source preconditions) -> persistence result
```

`ValidateFixPlan` accepts a plan from any producer: a built-in provider, a policy provider, an LSP client, a reviewer or an agent. A plan that names the findings it claims to resolve is checked against those findings. A plan without target findings can receive a validation-passed result, but cannot count as a verified repair. The caller supplies permitted source/edit scope and policy authority independently of the proposal. Producer-supplied classification or provenance does not grant authority. Suppressions, removal of failing operations and contract-changing constraint edits require explicit selection; disappearing diagnostics alone do not prove a repair. Freeze the ruleset, effective options, policy and referenced-source bytes for both lint passes. Every plan passes the same scope, completeness and output checks.

A `FixPlan` should contain:

- Source URI and content hash/version; format and dialect.
- Rule ID, diagnostic code, finding identity and provider version.
- Classification, explanation and explicit policy provenance.
- Ordered edits with source-relative JSON Pointer tokens and expected old values.
- Ordered atomic edit groups, conflicts and declined/unsupported reasons. Start with whole-plan rejection on conflict; no dependency graph scheduler.
- Source hashes and ordered edits sufficient to derive a preview. Candidate bytes, after-hashes and validation outcomes belong in the verification result, not mutable plan fields.

### Edit format

Data edits use standard [JSON Patch (RFC 6902)](https://www.rfc-editor.org/rfc/rfc6902) semantics: operations execute in order against the result of the previous operation. Internally, the planner can retain original node identities for conflict detection, but it must compile them into correct sequential pointer operations before export or application. Never reinterpret standard array indices as original-snapshot indices.

A versioned Vacuum plan envelope carries the source content hash, expected-absence conditions and policy provenance. A standard `test` checks an existing value; it cannot assert that a missing member is absent. Check absence against the snapshot, then apply the complete patch privately. Use `test` where it expresses a useful existing-value precondition; do not require an old value for every `add`.

Sequence order is part of JSON data: tag sorting can use standard `move` operations. YAML key retagging, mapping order and duplicate mapping-key occurrences require source-node identity outside the JSON data model. Keep these presentation edits in a separate, explicitly versioned envelope field, not custom JSON Patch opcodes. Implement only the concrete edit kinds needed by shipped providers. A consumer that lacks support for a required presentation edit rejects the complete plan. Do not add a generic extension-operation framework.

Defer automatic Overlay export. The existing `vacuum apply-overlay` command remains available for separately authored OpenAPI transformations. The required initial clients can consume the Vacuum plan and candidate bytes; an additional export translation is not a prerequisite. Add export only for a demonstrated consumer, with equivalence tests for every supported edit.

Return a plan and a separate verification result containing final findings, verified repair identities and declined/conflicting reasons. File writing returns its own success or error. Derive displayed status from these results; do not add a persisted or mutable six-state repair lifecycle. Retain the original finding until verification succeeds. In a CLI in-place workflow, report durable success only after writing succeeds. An in-memory/WASM caller can report a verified document without implying disk persistence.

Providers should be small and shared by operation: typed deduplication, metadata insertion, stable sorting, key retagging, scoped property updates. Select them by rule plus diagnostic code through one direct built-in lookup owned by the existing rule/function layer. Keep domain guards next to the domain functions. Share schema operations through the existing schema-check layer so OpenAPI and JSON Schema do not diverge.

Keep `AutoFixFunction` for compatibility. Its adapter must specify the existing in-place mutation contract, detect empty changes and validate the result. Decide separately whether to support returned replacement nodes; never silently change old plugin semantics. Legacy callbacks receive live rule context today. They are trusted plugin code, not an isolation boundary. Built-ins should receive a read-only planning context and produce edits instead of mutating indexed models.

### Execution sequence

1. Parse and lint an immutable snapshot using existing format/ruleset selection. Preserve ignores, severities, timeout and panic controls.
2. Ask providers for plans for selected, non-ignored findings. An error, missing source, incomplete validation or failed precondition means a refusal, not success.
3. Keep each producer plan ordered. Deduplicate identical shared-source edits and reject overlapping plans that need reconciliation. Order independent plans deterministically; do not reorder operations within a JSON Patch. Compile built-in node identities into sequential pointers. Do not build a general dependency scheduler or automatically resolve conflicting choices.
4. Apply the selected batch to one private document graph per owned source. Do not clone a complete document for each finding or each rule. Serialize without publishing mutations to the active Doctor/index graph.
5. Check the edits against caller-approved scope and the selected provider/policy contract, then reparse and rebuild models/indexes from the candidate bytes. Reuse the frozen referenced-source bytes and lint configuration; do not refetch mutable remote content for the second pass. Verify the targeted diagnostic is gone and no new diagnostics were introduced. Match stable identities; line numbers can move. Retain `Path`/`Paths` semantics. Required companion edits, such as a new tag description, belong in the same plan.
6. Reject unsupported or incomplete verification. A full parse or lint error must not produce a “fixed” file. Use a bounded number of explicit rounds; default to one batch and one verification pass, not a loop until counts fall.
7. For CLI writes, verify source hashes again, write same-directory temp files, preserve applicable permissions, and rename atomically per file. This prevents partial-file publication and detects observed stale input; a check followed by rename is not a filesystem compare-and-swap against an uncooperative concurrent writer. Document that limit; use versioned edits in LSP and explicit output files when concurrent editing cannot be coordinated. For an external multi-file plan, define rollback/recovery explicitly; individual renames are not a cross-file atomic transaction. Start with root-only writes and decline external edits until this is implemented.
8. Return final findings, verified repair identities, declined reasons, conflicts and any write errors. A validation-only external proposal is not included in repair counts. Preserve exit codes: 0 clean, 1 remaining threshold violations, 2 input/tool failure. Counts should describe the verified final document.

JSON must remain parseable JSON; YAML must retain comments, scalar styles, anchors, aliases and key ordering where edits permit. If the current encoder cannot preserve an important presentation property, expose that in preview and decline default automatic application. Do not promise byte-for-byte preservation from a node marshal.

## Surfaces and scope

Use one planner for Go library, CLI, reports, LSP and browser callers. Keep filesystem/network code outside the planner so the WASM agent can consume it without introducing platform dependencies. No WASM changes belong in this branch.

Proposed CLI behavior: explicit configured policy enables P edits; a dry-run/plan option emits diff plus structured disposition data. Review edits need explicit selection. Preserve the existing `--fix-file` contract.

Built-in providers change what `--fix` does for existing users. Today `--fix` runs only plugin callbacks, and CI jobs that already pass it would start receiving built-in edits on upgrade. Gate built-in providers behind a separate opt-in (a flag or a ruleset setting) for at least one release, and announce any later change to the `--fix` default in the release notes. Define stdin/stdout and multi-input behavior before adding flags: no source overwrite for stdin, machine stdout contains only the selected output, and one `--fix-file` cannot receive multiple inputs.

Reports and docs can show plans but should not write source files as a side effect. LSP edits must carry document versions and reject stale buffers. Browser calls return repaired bytes and a preview for the user to download. No automatic remote writes. Custom JavaScript functions do not gain filesystem or private-network authority through repair support.

Policy values must be scoped by document and schema/operation pointer, with ambiguous assignments rejected. Validate policy before any mutation. Keep provenance with the plan so an agent can explain exactly why a limit, license, response or security scheme was chosen. A language-model proposal enters through `ValidateFixPlan` like any other plan; it must not bypass these gates.

### Policy file (sketch)

67 rule IDs have policy-driven repair candidates. Start with exact values for contact and license plus an operation-description catalog. Prove those consumers before adding policy syntax. The following is a design sketch; diagnostic names must match the codes added at the actual producers:

```yaml
# vacuum-fix-policy.yaml, passed with --fix-policy
version: 1
policies:
  - id: org-contact                      # recorded as provenance in every plan it supplies
    rule: info-contact
    diagnostic: contact-missing
    scope:
      source: "./specs/payments.yaml"    # relative to this policy file
      pointer: "/info"
    value:
      name: Payments Platform
      email: payments-api@example.com
      url: https://example.com/payments-support
  - id: descriptions-catalog
    rule: operation-description
    diagnostic: description-missing
    scope:
      source: "./specs/payments.yaml"
      pointer: "/paths/~1payments/get"
    catalog: "./docs/operation-descriptions.yaml"   # source + operationId; missing keys decline
```

Rules for the format:

- A policy names the rule and diagnostic it answers. There are no wildcard rules.
- For the first policy providers, require an exact source and an exact target pointer. Match decoded JSON Pointer tokens, not string prefixes. Reject overlapping policy assignments for the same rule, diagnostic and target. Source and catalog paths resolve relative to the policy file. Catalog keys include source identity and operationId; duplicate or absent operation IDs decline lookup. Globs, inherited policies and pointer-prefix precedence are deferred until a concrete use case requires them.
- Missing catalog entries decline the repair. They never fall back to placeholder text.
- The policy file is validated against a schema before planning. An invalid policy fails the run with exit code 2.
- Start with one local policy file. No remote policy loading, ruleset embedding or `extends` behavior in the first version. These are not prerequisites for the existing 67 policy classifications.

## Implementation sequence and commit boundaries

1. **Engine hardening (standalone bug fix).** Detect no-op document transactions and report them as not fixed; define the callback return-value contract; preserve JSON output; source hash check and atomic root writes. Carry timeout/panic guards forward. No new public types; ships without the rest of this plan.
2. **First diagnostic identities and plan contract.** Add codes only for the first target cases, a versioned plan envelope, standard sequential JSON Patch and the concrete source-presentation edits those cases need. Use the existing rule/function registration ownership with a direct built-in planner lookup; no second capability registry. Derive available repairs from that lookup. Preserve existing result fields and plugin interfaces.
3. **Plan verification.** Separate planning and application from lint workers; add conflict detection, reparsing and relinting. Expose `ValidateFixPlan` for externally produced plans and a dry-run that emits plans, diffs and dispositions.
4. **First automatic providers.** Ship typed enum deduplication, tag ordering/deduplication, scheme deduplication, response-key quoting and required-path flags in focused commits, behind the built-in opt-in. Each provider gets positive, refusal, preservation and idempotence fixtures.
5. **Policy providers and their file schema.** Start with one local, exact-scope policy file for contact/license/documentation catalogs and ID naming. Add confirmed responses, endpoints, schema limits and security declarations after scoped policy and provenance tests.
6. **Remaining guarded providers.** Exact duplicate definitions, self-dependencies and supported const/enum redundancy. Singleton combinator rewrites only after reference/scope equivalence tests pass.
7. **All format and client surfaces.** Add JSON Schema CLI fixes and source-aware engine diagnostics, then LSP edits and structured report plans. Coordinate the planner API with the separate WASM work. Arazzo and AsyncAPI use their own source/reference semantics.
8. **Reviewed migrations and external edits.** Rename graphs, endpoint/topic changes, authentication migrations and explicit pruning. Add multi-file source ownership, recovery and stale-edit protection before write support. Until then, write support is root-only and external edits are declined.

Each stage must leave one working path. The plan executor replaces the old in-rule mutation path as the common coordinator; legacy callback signatures remain as adapters. Do not permanently maintain separate CLI, plugin and external-plan verification engines.

No estimates of development time are included. Provider counts understate the work: verification, source fidelity and multi-document ownership are the main engineering costs.

## Tests and performance gates

Every provider must prove:

- Minimal positive fixture: exact expected edit and targeted diagnostic removed.
- Negative fixture: ambiguous or unsupported case remains unchanged with a clear refusal.
- Already-clean fixture: no edit; a second repair pass is byte-stable.
- Preservation fixture: unrelated fields, comments, anchors/aliases and typed values survive; all incoming reference locations remain correct.
- Format/dialect fixture: JSON and YAML, applicable OpenAPI/AsyncAPI/JSON Schema versions; no accidental migration.

Engine tests must cover no-op document transactions, sibling-only callback edits, callback return semantics, sequential patch indices, absent-member additions, unsupported presentation edits, suppression/deletion proposals, unchanged verification configuration, partial mutation then error/panic, cancellation while waiting/working, conflicting providers, concurrent lint workers, stale disk/LSP sources, symlinks, unreadable/unwritable paths, temp-file/write failures, aliases, shared external refs, incomplete validation, suppressed findings, repeated diagnostic targets, and serialization failure. Run race checks on concurrency-sensitive packages. Use fuzz/property tests for node edits, typed equality, JSON Pointers and alias graphs; do not rely only on example counts.

Use `~/pb33f/demo/speed-test` for performance work, with the exact same toolchain, rules and inputs for before/after runs. Measure plain lint, plan-only, fix plus verification, and second-pass no-op separately. Record elapsed time, allocations, peak RSS, bytes changed, findings before/after and repairs per provider. Include a synthetic high-duplicate fixture and a shared-reference/alias fixture. Existing benchmark specs may contain few eligible repairs and cannot alone measure provider throughput.

The intended cost model is one original lint, indexed proposal generation, one private edit graph per source, and one verification lint. Deduplication should use typed hashes with equality checks on collisions; ordering should be stable O(n log n). Avoid per-finding full-tree searches and O(finding_count × document_size) clones. Do not claim a speed or memory improvement before measuring it. Plain lint with fixes disabled must retain its existing behavior and avoid constructing repair graphs.

## Validation performed for this audit

Run from the audit worktree root:

```sh
GOWORK=off GOTOOLCHAIN=go1.26.10 GOCACHE=/tmp/vacuum-benchmark-go-cache go run ./design/autofix/check
GOWORK=off GOTOOLCHAIN=go1.26.10 GOCACHE=/tmp/vacuum-benchmark-go-cache go run ./design/autofix/probe
GOWORK=off GOTOOLCHAIN=go1.26.10 GOCACHE=/tmp/vacuum-benchmark-go-cache go test ./motor ./cmd ./plugin -run 'AutoFix|Autofix|FixFile|FixWarns|PluginFix' -count=1
git diff --check
```

The inventory check passed for all 157 IDs. Focused existing tests passed in motor, cmd and plugin. The behavior probe confirmed the three cases above. The test selection is narrow; it is not the full suite, race coverage, performance evidence or proof that the proposed fixers work. This branch changes audit documents and audit utilities only. No dependency, Go version, production rule, WASM or website changes are made.
