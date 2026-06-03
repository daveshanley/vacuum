# AsyncAPI Implementation Plan

Sources:

- Spectral's current `packages/rulesets/src/asyncapi/index.ts` as rule-policy input.
- AsyncAPI `3.1.0` as the spec target.

vacuum is adding AsyncAPI `3.x` support only.

## Locked Decisions

- `vacuum lint` auto-detects AsyncAPI invisibly.
- AsyncAPI `2.x` is unsupported and exits with code `2`.
- Default ruleset is `asyncapi-recommended`.
- Custom rulesets run alone, format-filtered; vacuum does not auto-add `asyncapi-recommended`.
- Preserve Spectral v3 rule IDs exactly.
- Drop v2-only rules.
- Add vacuum-native v3+ rules as `asyncapi-*`, not `asyncapi-3-*`.
- Build inside vacuum first with `../libasyncapi`; refactor to Doctor later.
- Use `go.work` for development across sibling repos; release/tag lift happens after the stack works.
- `bundle`, `docs`, and `apply-overlay` remain OpenAPI-only and cleanly reject AsyncAPI.

## Phase 0: Workspace And Dependency Shape

Develop through `go.work` with local modules:

- `/Users/daveshanley/pb33f/vacuum`
- `/Users/daveshanley/pb33f/libasyncapi`
- `/Users/daveshanley/pb33f/libopenapi`
- `/Users/daveshanley/pb33f/doctor` if existing vacuum compile paths need it

Point the workspace at the same `libopenapi` line vacuum targets now, so any `libasyncapi` API drift shows up during development. Later release lift is explicit: update `libasyncapi` module deps, tag it, then add the vacuum require.

## Phase 1: Detection And Routing

Detection should happen before OpenAPI parsing.

Add AsyncAPI format constants and matching:

- `asyncapi3` as the `3.x` family
- `asyncapi3_0`
- `asyncapi3_1`

Routing must exist in both places:

- command/shared build helpers for default ruleset selection
- `motor.ApplyRulesToRuleSetWithOptions` as the hard guard for direct callers like dashboard, reports, and LSP

Do not rely only on `lint_cmd.go`.

## Phase 2: Motor AsyncAPI Branch

In AsyncAPI mode, skip OpenAPI construction completely:

- do not build `libopenapi` OpenAPI models
- do not build Doctor `DrDocument`
- leave OpenAPI `Document` / `DrDocument` nil
- build and populate AsyncAPI context instead

This prevents malformed OpenAPI state and null-deref failures.

Add AsyncAPI context through:

- `RuleSetExecution`
- `RuleSetExecutionResult`
- internal `ruleContext`
- `model.RuleFunctionContext`

Prefer `AsyncContext *asyncapi.Context` over a bare document. It should expose raw bytes, root node, `libasyncapi.Document`, high model, low model, version, index/rolodex, reference helpers, and location helpers.

## Phase 3: Rule Authoring Pattern

AsyncAPI built-ins are model-walking rules.

Default pattern:

- `given: "$"`
- function walks `libasyncapi` high model or visitor
- line/column/path comes from low nodes
- JSONPath `given` is allowed only for simple raw-node checks

Parity means same Spectral ID and same intent, not copied Spectral JSONPath.

## Phase 4: Ruleset Wiring And Safety

Add:

- `asyncapi-recommended`
- `extends: [[vacuum:asyncapi, recommended]]`
- likely `vacuum:asyncapi`
- likely `vacuum:asyncapi, all`
- `off` behavior matching existing ruleset patterns

Audit format filtering. Empty `Formats` rules currently run everywhere, so every AsyncAPI rule must carry AsyncAPI formats, and existing empty-format built-ins need review before AsyncAPI enters the shared pipeline.

## Phase 5: Document Validation

Verify or add in `libasyncapi`:

- unresolved document support
- resolved document support
- reference error collection
- 3.1.0 fixture/model coverage
- low-node location fidelity

Emit structural/document errors as lint results when possible:

- `asyncapi-3-document-resolved`
- `asyncapi-3-document-unresolved`

Only hard tool/parser failures exit `2`.

## Phase 6: Spectral V3 Parity

Implement the current v3-applicable Spectral rules: the `asyncapi-3-*` rules plus the bare-prefix shared rules that apply to v3. Keep `asyncapi-3-operation-security`; it exists in current source.

Do not implement any v2-only rule.

## Phase 7: vacuum-native rules

Add spec-semantic rules Spectral misses:

- `asyncapi-server-variables`
- `asyncapi-server-security`
- `asyncapi-operation-channel`
- `asyncapi-operation-messages`
- `asyncapi-operation-reply`
- `asyncapi-message-examples`
- `asyncapi-unused-components`
- `asyncapi-content-type`

For `asyncapi-unused-components`, sweep reusable maps explicitly: schemas, servers, channels, operations, messages, security schemes, server variables, parameters, correlation IDs, replies/reply addresses where modeled, operation traits, message traits, and server/channel/operation/message bindings.

Binding/protocol consistency can wait unless it stays small.

## Phase 8: Schema Validation Spike

Before implementing schema rules, pin the dialect strategy.

The default AsyncAPI Schema Object is not OpenAPI 3.0 schema and not plain JSON Schema 2020-12. Decide the adapter/metaschema first, probably closer to vacuum's JSON Schema path than `libopenapi-validator`.

Runtime behavior:

- omitted `schemaFormat` uses AsyncAPI default for the document version
- supported JSON Schema-compatible formats are validated
- unsupported formats emit `info`
- instance validation failures emit normal rule results
- validator limitations do not panic or become tool failures

## Phase 9: Reports, Stats, LSP

Add AsyncAPI stats:

- channels
- operations
- messages
- servers
- schemas
- parameters
- replies
- security schemes
- bindings
- tags
- references

Update quality score so AsyncAPI document validation failures get the same heavy penalty as invalid OpenAPI schemas.

Wire and verify:

- lint
- report
- spectral-report
- html-report labels
- dashboard
- vacuum-report
- LSP diagnostics

Also check diff/original filtering and docs-diagnostics conversion so AsyncAPI results are not forced through Doctor v3 assumptions.

## Phase 10: Tests And Release Lift

Add fixtures for valid 3.1.0, unsupported 2.x, malformed 3.x, refs, server variables, operation/channel/message links, replies, security, schema formats, examples, unused components, stats, and quality score.

Focused checks:

```bash
go test ./cmd -run 'Lint|AsyncAPI'
go test ./motor -run 'AsyncAPI'
go test ./functions/asyncapi
go test ./rulesets
```

After workspace development is stable, do the release-shaped lift:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test ./...
```

That final check only makes sense after `libasyncapi` is added to vacuum's module graph via a real versioned dependency path.
