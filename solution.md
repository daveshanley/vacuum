# Issue 919 Race Condition Fix

This document describes the final staged fix for issue 919 on branch
`issue-919`.

## Problem

Issue 919 exposed a concurrency bug when OpenAPI example validation and other
schema rules operate on the same parsed document at the same time.

The failure shape was:

1. `oas3-valid-schema-example` validates an example for a schema that contains
   a local `$ref`.
2. The example rule builds or renders a libopenapi schema for
   `libopenapi-validator`.
3. That path can touch mutable YAML/schema internals.
4. Another rule, such as `schema-type-check`, reads the same shared model.
5. The shared model can be observed while it is being mutated, producing race
   detector failures and unstable schema results.

The issue was visible as referenced enum data being misread after example
validation ran, but the underlying problem was broader: validation must not
mutate shared Doctor/libopenapi schema state during a parallel lint run.

## Relevant Existing Guard

`parser/json_schema.go` already serializes its schema validation path with
`globalValidatorMu`. Its comment is important: YAML rendering/desolving can
mutate node metadata, so callers that validate the same parsed nodes
concurrently must be protected.

The OpenAPI examples rule was not using that guarded parser path. It created its
own `schema_validation.NewSchemaValidator()` and validated examples in a worker
pool, so it needed its own isolation boundary.

## Dependency Update

The branch upgrades the requested dependencies:

- `github.com/pb33f/libopenapi` to `v0.38.5`
- `github.com/pb33f/libopenapi-validator` to `v0.13.13`

The plugin sample module was updated to the same dependency versions so its
module graph stays aligned with the root module.

## Final Fix: OpenAPI Example Validation

`functions/openapi/examples_schema.go` now rebuilds the validation schema from a
private YAML node tree:

```go
root := utils.CloneYAMLNode(index.ResolveRefsInNode(sourceRoot, idx))
```

The important details are:

- `index.ResolveRefsInNode` resolves local `$ref` values inside the schema node,
  including refs nested under sequence keywords such as `allOf`, `oneOf`,
  `anyOf`, and `items`.
- Nested external refs are not inlined by this helper and still use the existing
  index/rolodex. That is an explicit remaining boundary, not a new regression.
- It uses the existing document index instead of building a new `SpecIndex` for
  every example validation.
- The resolved node is passed through `utils.CloneYAMLNode` before build. That
  final deep clone is intentional because the resolver may reuse scalar/key
  nodes, and the validator/rendering path has already shown that YAML node
  metadata can be mutable.
- The cloned node is then used to build a fresh low/high schema for validation.
- Repeated examples for the same schema reuse one cloned validation schema inside
  the worker handling that object. The cache is intentionally worker-local; it
  is not shared across rules or workers.

The previous staged synthetic-document approach was removed. That code walked
the whole source document to find schema paths, rebuilt partial YAML documents,
and created fresh spec indexes. It was both slower and incomplete for some
sequence-nested schema shapes.

## Why The Previous Staged Approach Was Replaced

The synthetic clone path had two major problems.

First, it was too expensive. It performed document-scale path discovery and
index construction while validating examples. On large documents this turned a
small schema validation into work proportional to the surrounding spec.

Second, it had a correctness hole. If the schema being validated was nested
inside a sequence, the path finder could fail and fall back to a cloned root with
the original shared index. That meant nested local refs could still route
through shared source state.

Using `index.ResolveRefsInNode` is smaller and directly matches the problem:
resolve the local ref closure inside the schema node, then build validation from
an isolated copy.

## Core Schema Rule Fix

`functions/core/schema.go` no longer mutates shared schema state with:

```go
schema.GoLow().Index = schemaIndex
```

It now calls `cloneSchemaForCoreValidation`. If the schema already uses the
requested index, it is reused. If not, the schema root is cloned and rebuilt with
the requested index. That keeps the validation context correct without writing
back into the shared schema.

## YAML Tag Stability

`functions/schemachecks/value.go` and
`functions/schemachecks/validators.go` now compare YAML tags through
`node.ShortTag()` instead of raw `node.Tag` values. This keeps checks stable
when equivalent YAML type tags use different long/short representations.

## External Ruleset Timeout Race

A separate race was found while running the race suite. External ruleset loading
previously started a goroutine and returned on timeout, but that late goroutine
could keep mutating shared ruleset/log state after the caller moved on.

The branch adds `rulesets/external_ruleset_loader.go`. The loader now:

1. Creates a timeout context.
2. Clones the current `RuleSet` into worker-owned state.
3. Loads external rulesets into that worker copy.
4. Buffers worker log records.
5. Copies a snapshot back only through controlled copy functions.
6. Flushes buffered logs from the caller goroutine.

`rulesets/remote_ruleset.go` also checks context around local/remote loads,
closes HTTP response bodies, and recalculates local/remote mode for nested
extends.

## Tests Added

The issue regression test is:

```go
TestSchemaType_Issue919_ConcurrentExampleValidationDoesNotPoisonRefEnums
```

It repeatedly builds the issue-shaped OpenAPI document and runs
`ExamplesSchema` and `SchemaTypeCheck` concurrently. It fails if schema type
checking observes the poisoned enum/type state.

The sequence-nested ref regression test is:

```go
TestCloneSchemaForExampleValidation_ResolvesSequenceLocalRefs
```

It validates that a schema cloned for example validation no longer contains a
local `$ref` under an `allOf` sequence. This covers the hole in the previous
synthetic-document approach.

The performance benchmark added for this code path is:

```go
BenchmarkCloneSchemaForExampleValidation_LargeLocalRefClosure
```

It builds a large document with 750 unrelated schemas before the target schema.
That fixture specifically exposes document-scale work in the old implementation.

The repeated-example benchmark is:

```go
BenchmarkExamplesSchema_ManyNamedExamplesSharedSchema
```

It validates 20 named examples against one media-type schema and proves repeated
examples reuse the worker-local validation schema clone.

The external-ruleset timeout regression test is:

```go
TestRuleSet_GetExtendsRemoteSpec_TimeoutDoesNotWaitForNonContextAwareClient
```

It proves a non-context-aware HTTP client cannot keep mutating returned ruleset
state after the timeout path returns.

## Performance Profile

The benchmark was run sequentially against:

- the staged synthetic-document implementation exported to
  `/tmp/vacuum-issue919-heavy`
- the current resolver-based worktree

Command shape:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test ./functions/openapi \
  -run '^$' \
  -bench BenchmarkCloneSchemaForExampleValidation_LargeLocalRefClosure \
  -benchmem \
  -count=5 \
  -cpuprofile /tmp/vacuum-issue919-<variant>.cpu.pprof \
  -memprofile /tmp/vacuum-issue919-<variant>.mem.pprof
```

Results:

```text
synthetic-document implementation:
  183987-202730 ns/op
  900966-900987 B/op
  4931 allocs/op

resolver-based implementation:
  5425-5476 ns/op
  18499 B/op
  154 allocs/op
```

Mean runtime changed from about `193847 ns/op` to about `5451 ns/op`, a
`35.6x` speedup on this large fixture. The important performance story is that
the cost moved from document-scale path discovery to schema-closure work.
Allocation bytes dropped about `48.7x`, and allocation count dropped about
`32.0x`.

The old memory profile was dominated by the removed machinery:

```text
findExampleSchemaPathWithTrail             17.9 GB flat
libopenapi/index.addNodeLineEntry          14.4 GB flat
cloneSchemaDocumentForExampleValidation    18.6 GB cumulative
```

The new memory profile is dominated by the expected isolated schema work:

```text
libopenapi/utils.cloneYAMLNode              7.26 GB flat
cloneSchemaForExampleValidation            19.1 GB cumulative
libopenapi/index.ResolveRefsInNode          1.96 GB cumulative
low/base.(*Schema).Build                    3.07 GB cumulative
```

The removed path-finding and per-validation `SpecIndex` construction no longer
appear in the new hot path.

The repeated-example benchmark was run against the resolver implementation
before and after the worker-local cache:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test ./functions/openapi \
  -run '^$' \
  -bench BenchmarkExamplesSchema_ManyNamedExamplesSharedSchema \
  -benchmem \
  -count=5
```

Results:

```text
resolver path without worker-local cache:
  562431-586761 ns/op
  1089240-1089663 B/op
  9193-9194 allocs/op

resolver path with worker-local cache:
  255208-271369 ns/op
  244134-244229 B/op
  3229 allocs/op
```

Mean runtime changed from about `573169 ns/op` to about `261919 ns/op`, a
`2.2x` speedup for 20 named examples. Allocation bytes dropped about `4.5x`,
and allocation count dropped about `2.8x`.

Generated profiles:

- `/tmp/vacuum-issue919-heavy.cpu.pprof`
- `/tmp/vacuum-issue919-heavy.mem.pprof`
- `/tmp/vacuum-issue919-new.cpu.pprof`
- `/tmp/vacuum-issue919-new.mem.pprof`

## Validation

Focused correctness passed:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test ./functions/openapi \
  -run 'TestCloneSchemaForExampleValidation_ResolvesSequenceLocalRefs|TestSchemaType_Issue919_ConcurrentExampleValidationDoesNotPoisonRefEnums|TestSchemaType_Issue916_AllOfRefDefaultDoesNotPoisonSiblingEnums|TestExamplesSchema' \
  -count=1
```

Parser/core focused validation passed after removing the duplicate
`low.BuildModel` call from `parser.ConvertNodeIntoJSONSchema`:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test ./parser ./functions/core
```

Focused race validation passed:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test -race ./functions/openapi \
  -run 'TestCloneSchemaForExampleValidation_ResolvesSequenceLocalRefs|TestSchemaType_Issue919_ConcurrentExampleValidationDoesNotPoisonRefEnums|TestSchemaType_Issue916_AllOfRefDefaultDoesNotPoisonSiblingEnums|TestExamplesSchema' \
  -count=1
```

Broad validation and staged whitespace checks passed:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test ./...
GOWORK=off GOCACHE=/tmp/go-build go vet ./...
git diff --cached --check
```

## Current Tradeoff

The resolver-based fix intentionally clones the resolved schema node before
building the validation schema. That costs some allocation, but it is the right
tradeoff here: it keeps validation off shared YAML nodes while avoiding the much
larger cost of path discovery and new spec indexes per example.
