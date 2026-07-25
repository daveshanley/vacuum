# Issue #937: external-file model collection must be source-aware

## Status

Proposed implementation plan.

Issue: https://github.com/daveshanley/vacuum/issues/937

## Decision

Fix the model-identity bug in `github.com/pb33f/doctor`, release Doctor, and then update Vacuum to the fixed Doctor version with a Vacuum-level regression test.

Do not add an `oas3-valid-schema-example` workaround. The rule correctly validates every media type present in `DrDocument.MediaTypes`; Doctor silently removes distinct media types when they occupy the same line and column in different external files.

The Doctor fix must apply to all five collections using the same source-blind identity:

- schemas;
- skipped schemas;
- parameters;
- headers; and
- media types.

This is a correctness fix with a performance constraint. Collection identity must include the source index without introducing path-string allocation or a second document walk.

## Reproduced behavior

The issue fixture was reproduced against the current Vacuum checkout with release-shaped module resolution:

```bash
GOWORK=off GOCACHE=/tmp/go-build go run . lint -d <fixture>/openapi.yaml --no-style
GOWORK=off GOCACHE=/tmp/go-build go run . lint -d <fixture>/openapi-bundled.yaml --no-style
```

| Input | `oas3-valid-schema-example` violations |
| --- | ---: |
| Split, both external media types at the same line and column | 1 |
| Bundled equivalent | 2 |
| Split, with one blank line added to the second external response | 2 |

The blank-line control changes no OpenAPI semantics. It only changes the second media type's line number, which proves that the missing violation is caused by source-blind positional deduplication.

## Root cause

Doctor's `model/walk_model.go` currently creates a packed `uint64` key from only a node's line and column:

```go
func createKey(line, column int) uint64 {
    return uint64(line)<<32 | uint64(column)
}
```

That key is used by the collector state for `Schemas`, `SkippedSchemas`, `Parameters`, `Headers`, and `MediaTypes`.

Line and column identify a node only inside one YAML document. They are not globally unique across the root document and every file in the rolodex. In issue #937, both external response files have their media type at the same line and column, so the second media type is treated as another occurrence of the first and is discarded.

Vacuum later loops over `ruleContext.DrDocument.MediaTypes` in `functions/openapi/examples_schema.go`. By that point the second external media type no longer exists in the Doctor collection, so the rule cannot validate its example.

This is not:

- a failure to resolve the transitive schema `$ref`;
- validator state leaking between examples;
- example-validation concurrency;
- shared-schema caching in Vacuum; or
- a difference in the logical OpenAPI document.

## Goals

- Preserve distinct Doctor models when their YAML coordinates match but their source indexes differ.
- Continue deduplicating repeated walks of the same source object.
- Preserve canonical-parent selection when one source object is reachable through multiple reference paths.
- Correct all five affected collection states: schemas, skipped schemas, parameters, headers, and media types.
- Make split and bundled forms produce the same example-validation findings.
- Preserve deterministic output in cached, uncached, synchronous, and worker-pool walks.
- Avoid new per-object path-string allocations and avoid an additional traversal.
- Prove the fix at both the Doctor collection layer and Vacuum's public lint behavior.

## Non-goals

- Rewriting Doctor's walker or schema cache.
- Moving example traversal out of Doctor and into Vacuum.
- Changing the meaning, severity, or default status of `oas3-valid-schema-example`.
- Reporting the same external definition once per `$ref` use site. Doctor should continue to collect one definition and retain its multiple paths.
- Changing libopenapi reference resolution.
- Refactoring all line-number-indexed Doctor structures. Structures that use line as a lookup bucket and then filter by source are not identity maps and do not need to change for this issue.
- Adding a permanent `replace` directive to Vacuum or publishing a Vacuum module that depends on an unreleased Doctor revision.

## Required invariants

The implementation must distinguish these cases:

| Case | Required collection behavior |
| --- | --- |
| Same source index, same line and column, reached repeatedly | One collected object |
| Same source index, same line and column, reached through different `$ref` paths | One collected object with existing canonical-parent behavior |
| Different source indexes, same line and column | Two collected objects |
| Different source indexes and different coordinates | Two collected objects |
| Source index unexpectedly unavailable | Do not collapse unrelated objects solely because coordinates match |

The last row is important: a missing source identity must fail open for collection correctness, not silently recreate the false-negative.

Failing open has its own failure mode: if source identity resolves for one occurrence of an object but not for another, the two keys differ and the object is collected twice, which surfaces as duplicate lint findings — the opposite failure from issue #937. If the fallback path is reachable at all, the same-source-reached-twice test must also run through a shape that exercises it, and the walk-mode matrix must confirm no mode systematically fails resolution.

## Phase 1: add failure-first Doctor coverage

Add a minimal multi-file fixture under Doctor's `test_specs/`, scoped to this bug. The fixture should retain the issue's important shape:

```text
test_specs/issue-937/
├── openapi.yaml
├── responses/
│   ├── first.yaml
│   ├── second.yaml
│   └── schemas/
│       └── base.yaml
└── ...
```

`first.yaml` and `second.yaml` must deliberately place their collected objects at identical line and column coordinates. Add comments in the fixture and test warning that whitespace is part of the regression setup.

Add focused tests near `model/walk_model_test.go`:

### 1. Distinct sources with equal coordinates

Build the multi-file model with:

```go
datamodel.DocumentConfiguration{
    BasePath:            fixtureDir,
    SpecFilePath:        rootPath,
    AllowFileReferences: true,
}
```

Assert that both external media types are present. Verify their source indexes or absolute source locations differ, rather than checking only the slice length.

The old Doctor version must fail this test by returning one media type.

### 2. Same source reached more than once

Reference the same external response definition from at least two response codes. Assert that the media type is still collected once and that its generated path/canonical-parent behavior remains stable.

This prevents the source-aware fix from turning use-site aliases into duplicate model entries.

### 3. Shared collector identity

Add production-shaped multi-file coverage so the same-coordinate/different-source case passes through the real event emission and collection call sites for:

- schema;
- parameter;
- header; and
- media type.

Add one focused circular-reference fixture for skipped schemas. Its reference/use-site nodes must occupy identical coordinates in different external files so the test exercises the exact `SkippedSchemaChan` identity boundary.

The issue's response fixture can remain focused on media types. Use a second compact Doctor collector fixture if putting all five shapes into the issue reproduction would make it harder to understand.

Direct key/collector unit tests are useful for the identity truth table, but they are not sufficient on their own: they cannot prove that each production emitter supplies the correct node and source identity. Every affected collection must therefore have at least one real multi-file assertion.

### 4. Walk-mode matrix

Run the distinct-source and repeated-source assertions through:

- cached/default walking;
- uncached synchronous walking; and
- uncached worker-pool walking.

The same objects and counts must be produced in every mode. Repeat the worker-pool case enough times to expose nondeterministic replacement or collection races.

## Phase 2: make Doctor collection identity source-aware

Candidate A is the preferred implementation. Candidate B is a fallback only if the stability check disproves node-pointer identity for any production walk mode or collection.

### Candidate A: node-pointer identity

The collector loop already holds the collected object's `*yaml.Node` (`SchemaNode`, `ParamNode`, `HeaderNode`, `MediaTypeNode`) when it builds the key. Key the state maps on that pointer directly:

- each rolodex file is parsed once, so objects in different files yield different node pointers even at identical coordinates;
- repeated reaches of one definition resolve to the same parsed node, so deduplication is preserved;
- the key is a single comparable pointer with zero per-object interface assertions and zero allocation.

This is strictly cheaper than a composite key. It is only valid if node pointers are stable: no walk mode may re-emit a cloned or synthetic node for an already-collected object. That is proven or refuted by the stability check.

### Candidate B: source-position fallback

Replace the coordinate-only collector key with a comparable source-position key:

```go
type sourcePositionKey struct {
    source   *index.SpecIndex
    position uint64
    fallback *yaml.Node
}
```

The exact names may change, but the semantics must remain:

- `source` is the `SpecIndex` that owns the exact emitted node;
- `position` retains the existing packed line/column value;
- `fallback` is nil when `source` is known and uses node identity only when the source index is unavailable.

The source and position must describe the same identity domain. Do not obtain `source` from the collected model's resolved value while taking `position` from the emitted event node.

That distinction matters for skipped circular schemas:

- `WalkedSchema.SchemaNode` comes from `schemaProxy.GetSchemaKeyNode()`, which represents the reference/use site;
- `WalkedSchema.Schema.Value` is the resolved target schema; and
- the resolved target's `SpecIndex` can belong to a different file from `SchemaNode`.

Combining the resolved target index with the use-site coordinates is not a valid source-position key. Two different external use sites at identical coordinates could still collide when they resolve into the same target index.

If candidate B is required, attach the owning index at event emission time:

```go
type WalkedSchema struct {
    Schema      *Schema
    SchemaNode  *yaml.Node
    SourceIndex *index.SpecIndex
}
```

Add the equivalent field to `WalkedParam`, `WalkedHeader`, and `WalkedMediaType`.

Each emitter must populate `SourceIndex` from the same low-level object that supplied the emitted node:

- schemas and skipped schemas: the emitting `schemaProxy.GoLow().GetIndex()`;
- parameters: `param.GoLow().GetIndex()`;
- headers: `header.GoLow().GetIndex()`; and
- media types: `mediaType.GoLow().GetIndex()`.

The collector then combines `event.SourceIndex` with `event.*Node.Line` and `event.*Node.Column`. It must not call `FindNodeOriginWithValue` for every object and must not build or hash absolute path strings in the hot collector loop.

Candidate B's `fallback` field still depends on node-pointer stability when an owning index is unavailable. If that branch is reachable, test it explicitly with repeated emissions of the same source object so the fallback cannot create duplicate collection entries.

Whichever candidate wins, change the collector state maps from `map[uint64]int` to the new key type (`map[*yaml.Node]int` for candidate A, `map[sourcePositionKey]int` for candidate B) for:

- `skippedSchemasState`;
- `seenSchemasState`;
- `seenParametersState`;
- `seenHeadersState`; and
- `seenMediaTypesState`.

Keep `collectFoundational`'s existing replacement rule. When the chosen identity key is identical, `drV3.CompareByParentPosition` should continue selecting the canonical occurrence. When only the coordinates match but the source differs, append the new object.

Do not use:

- absolute path concatenated with line and column;
- `fmt.Sprintf`;
- node hashing;
- filename hashing with collision risk;
- YAML content hashes, because two distinct files may intentionally contain identical content; or
- the expanding generated JSONPath, because use-site paths are not definition identity.

### Stability check

Before finalizing either key, prove in Doctor tests:

1. **Node-pointer stability:** repeated references to one external definition, in every walk mode (cached, uncached synchronous, worker-pool), re-emit the same `*yaml.Node` pointer — never a clone or synthetic node. If this holds, use candidate A.
2. **Source-index stability:** repeated emissions of one node reuse the same owning `*index.SpecIndex`. Required only for candidate B; if libopenapi can produce multiple indexes for one physical or remote source, use a stable rolodex-owned source identifier instead. Do not fall back to source path strings without measuring the effect.

### Fallback cost: attach the index at emit time

The walker knows the owning `SpecIndex` when it emits `WalkedSchema`/`WalkedParam`/`WalkedHeader`/`WalkedMediaType` events. Candidate B therefore widens those public context structs and every emitter even though Candidate A needs no event-shape change. This is acceptable only as the correctness-preserving fallback if the stability check invalidates Candidate A.

## Phase 3: audit adjacent source-blind identity

Search Doctor for other maps or caches that use only line, column, or a packed coordinate as global identity.

Classify each occurrence:

- **Identity map:** must include source identity or have a documented proof that its scope is one index.
- **Lookup bucket:** may remain line-based if every read subsequently filters by source index and node identity.
- **Graph/render-only state:** leave unchanged unless the issue fixture exposes incorrect behavior.

In particular:

- `lineObjects` may remain a line bucket only if `LocateModelsByKeyAndValue` continues filtering candidates by their `SpecIndex`;
- `nodeValueMap` should be checked for multi-file graph collisions, but graph behavior should be changed only with a focused failing test;
- canonical-path and schema-cache keys must not be broadened as part of this issue unless the new tests demonstrate the same collision.

Record any newly discovered correctness bug separately instead of turning issue #937 into an unbounded Doctor identity rewrite.

## Phase 4: release Doctor and integrate Vacuum

1. Run Doctor's focused tests and full suite.
2. Tag a new Doctor patch release containing the source-aware collector.
3. Update only Doctor's version in Vacuum's `go.mod` and the resulting `go.sum` entries.
4. Do not add a committed `replace` directive.
5. During local cross-repository development, the existing workspace may point Vacuum at the Doctor checkout, but all final Vacuum verification must use `GOWORK=off` so the released module is what passes.

Before updating Vacuum, prove the Vacuum regression test fails against Doctor `v0.0.79`. After the dependency update, prove it passes with the released fixed version.

## Phase 5: add the Vacuum regression

Add a minimal fixture under:

```text
motor/test_data/issue_937/
```

Keep the issue's four-file split layout and add a bundled control. The two response files must keep their media types and invalid example values at identical coordinates.

Add:

```go
func TestRuleSetExecution_Issue937_ExternalResponseExamplesAtSameCoordinates(t *testing.T)
```

The test should:

1. load the split root bytes;
2. run `ApplyRulesToRuleSet` with `AllowLookup: true` and `SpecFileName` set to the fixture's root path;
3. filter results to `rulesets.Oas3ValidSchemaExample`;
4. assert exactly two `got string, want integer` findings;
5. assert one finalized result originates in `responses/first.yaml`;
6. assert one finalized result originates in `responses/second.yaml`;
7. assert the two result paths and source locations are deterministic; and
8. run the bundled control and assert the same two semantic findings.

Do not assert only the total result count. Origin assertions are necessary to prove that the second external object survived collection rather than that one example emitted duplicate validator errors.

Use the motor boundary because it exercises production Doctor construction and result-origin finalization. A direct `ExamplesSchema.RunRule` test does not populate `RuleFunctionResult.Origin`; duplicating the fixture at the function layer would not prove an additional boundary. Add a command-level test only if the motor test and final CLI reproduction disagree.

## Phase 6: performance and race verification

Candidate A's key is pointer-sized. Candidate B's key is larger than `uint64` and widens each emitted event with a source-index pointer. Either way, measure rather than assume the cost.

Capture Doctor before/after benchmark results with:

```bash
go test ./model -run '^$' \
  -bench 'BenchmarkWalker_Test(Burgers|Stripe|CountedSchemas)$' \
  -benchmem -count=10
```

Compare with `benchstat`.

Acceptance:

- no new per-collected-object allocation caused by source-key construction;
- no additional model walk or rolodex origin lookup;
- `allocs/op` must remain unchanged;
- any `bytes/op` increase must be explained by the chosen key representation and reviewed before acceptance;
- a statistically significant `ns/op` regression greater than 2% on Stripe or counted-schema fixtures blocks the change unless it is root-caused, documented, and explicitly accepted;
- any map-memory increase is limited to the expected larger identity key and documented;
- no race failures in cached or worker-pool modes.

If the source-position fallback causes a measurable regression, optimize the representation while keeping collision-free source identity. Do not restore coordinate-only identity for speed.

## Verification commands

### Doctor

Run the smallest tests first, then broaden:

```bash
go test ./model -run 'Issue937|SourcePosition|External.*Coordinate'
go test -race ./model -run 'Issue937|SourcePosition|External.*Coordinate' -count=20
go test ./model ./model/high/v3
go test ./...
```

Run the benchmark comparison described above.

### Vacuum

Use release-shaped dependency resolution:

```bash
GOWORK=off GOCACHE=/tmp/go-build \
  go test ./motor -run 'Issue937'

GOWORK=off GOCACHE=/tmp/go-build go test ./...
GOWORK=off GOCACHE=/tmp/go-build go build ./...
git diff --check
```

Re-run the real CLI reproduction:

```bash
GOWORK=off GOCACHE=/tmp/go-build go run . lint -d \
  motor/test_data/issue_937/openapi.yaml --no-style

GOWORK=off GOCACHE=/tmp/go-build go run . lint -d \
  motor/test_data/issue_937/openapi-bundled.yaml --no-style
```

Both commands must report two `oas3-valid-schema-example` violations.

## Acceptance criteria

- The unmodified issue #937 split fixture reports both invalid examples.
- The bundled control reports the same two semantic violations.
- Adding or removing harmless blank lines from either external response does not change the violation count.
- Distinct external files with identical YAML coordinates remain distinct in all affected Doctor collections.
- Production-shaped multi-file tests cover schemas, skipped schemas, parameters, headers, and media types through their real event emitters.
- Repeated references to one external definition remain deduplicated.
- Cached, uncached synchronous, and uncached pooled walks produce the same collection contents.
- Existing canonical path and multi-path behavior is unchanged for one definition with multiple use sites.
- Vacuum contains no issue-specific traversal workaround.
- Vacuum has no committed `replace` directive.
- Doctor and Vacuum focused, race, full test, and build checks pass.
- Benchmarks show unchanged `allocs/op`, no unexplained `bytes/op` increase, and no unaccepted statistically significant `ns/op` regression above 2%.
- The final Vacuum proof uses the released Doctor module with `GOWORK=off`.

## Delivery order

1. Doctor failing tests.
2. Doctor source-aware collection implementation.
3. Doctor focused, race, full-suite, and benchmark verification.
4. Doctor patch release.
5. Vacuum failing regression pinned to Doctor `v0.0.79`.
6. Vacuum Doctor dependency update.
7. Vacuum focused and full release-shaped verification.
8. Manual split-versus-bundled CLI proof.

This order makes the ownership boundary explicit and prevents a Vacuum release from depending on an unpublished local Doctor checkout.
