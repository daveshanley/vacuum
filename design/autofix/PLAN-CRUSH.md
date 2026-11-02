# Autofix plan-crush

Reviewed 2026-11-02 using `/Users/daveshanley/pf/skills/plan-crush/SKILL.md`. Scope: the user's updated autofix plan, the earlier review corrections, and only the repository contracts needed to test those proposals. Findings below describe the reviewed proposal; the replacements are applied to [README.md](README.md). This is a design review, not implementation proof.

## [P1] Original-snapshot indices would make standard patches edit different data

**Section / evidence:** Edit format; [RFC 6902, sections 4 and 4.6](https://www.rfc-editor.org/rfc/rfc6902). The proposed format used standard operation names but fixed array pointers to the original snapshot. It also required old-value tests for additions and classified sequence reordering as outside JSON.

**Problem:** A standard producer and the proposed applier could modify different array elements. Testing an absent member fails; it cannot express a missing-field precondition. Sequence moves already exist in the standard.

**Why it does not earn its complexity:** A second patch interpretation breaks the stated interoperability goal and adds a compatibility matrix without providing a required outcome.

**Smaller design:** Sequential RFC 6902 operations. Compile internal node identities into those operations. Put source hashes, expected absence and concrete YAML presentation edits in the versioned Vacuum envelope. No custom JSON Patch opcodes or generic extension registry.

**Outcomes preserved:** Precise edits, standard tooling, stale-source checks, YAML retagging and stable internal node tracking.

**Verify:** Remove array index 0 then replace index 1; add a missing member; move a tag; reject an unsupported required presentation edit. Compare standard data edits with an independent RFC-conforming applier.

**Confidence:** High. Encoding for duplicate YAML key occurrences still needs a concrete fixture before its provider ships.

## [P1] Passing lint does not establish that an external proposal repaired the contract

**Section / evidence:** External-plan validation and execution sequence; [inline ignore handling](../../motor/inline_ignore.go), [operation descriptions](../../functions/openapi/operation_descriptions.go), and [schema checks](../../functions/schemachecks/validators.go).

**Problem:** Removing an operation, suppressing its rule or weakening a constraint can remove a finding without introducing another one. A producer's claimed classification cannot authorize these changes. Mutable external references can also make the before/after comparison test different inputs.

**Why it does not earn its complexity:** A broad “verified” status would imply assurance that the lint pass does not provide. More status transitions cannot supply missing authority or semantic proof.

**Smaller design:** Caller-owned scope and selection, independent of the submitted plan. Verify the selected provider/policy contract and freeze lint settings plus dependency bytes. Keep validation-only proposals out of repair counts. Contract-changing or suppression edits require explicit selection; report exactly what was checked, not general API equivalence.

**Outcomes preserved:** First-class agent proposals, shared validation, policy provenance and honest results.

**Verify:** An external proposal that inserts an ignore, deletes the failing operation or changes a constraint cannot pass as an automatic repair. A proposal without target findings can return validation success but zero verified repairs. A changed remote reference cannot silently enter the second pass.

**Confidence:** High. The exact provider predicates must be proved per diagnostic during implementation.

## [P2] Target-only no-op detection would reject valid plugin repairs

**Section / evidence:** Standalone hardening; [callback signature](../../model/autofix.go), [transaction implementation](../../motor/auto_fix_transaction.go), [document-edit test](../../motor/auto_fix_transaction_test.go).

**Problem:** Callbacks receive the whole document and can edit a parent or sibling. An unchanged target is not an unchanged repair transaction.

**Why it does not earn its complexity:** A target-specific shortcut creates an incorrect parallel definition of a change. The transaction already tracks original and private document nodes.

**Smaller design:** Compare complete transaction changes, reusing the existing graph bookkeeping. Distinguish “an edit occurred” from “the finding was resolved”; verification establishes the latter.

**Outcomes preserved:** Existing plugin mutation behavior, no-op rejection, rollback and accurate repair counts.

**Verify:** No-op callback remains unfixed; sibling-only repair is not discarded; unrelated mutation does not qualify as a verified repair; errors and panics publish no partial edits.

**Confidence:** High. Return-node behavior remains an explicit compatibility decision before the standalone patch.

## [P2] A policy language would delay providers that need only exact values

**Section / evidence:** Policy format and implementation stage 2; [contact rules](../../functions/openapi/info_contact_properties.go), [license rule](../../functions/openapi/info_license.go), and [description rule](../../functions/openapi/operation_descriptions.go).

**Problem:** Globs, pointer-prefix specificity, ruleset embedding and inheritance were being settled before any policy provider existed. The example also omitted a required diagnostic and did not satisfy its own glob opt-in condition.

**Why it does not earn its complexity:** The first selected consumers need a known value for a known document location. No inspected consumer requires scope precedence, remote policy fetching or inheritance.

**Smaller design:** One schema-validated local file, exact source/target, rule/diagnostic and a value or source-qualified catalog key. Reject ambiguity. Resolve paths relative to the policy file. Design the schema alongside the first real consumers; carry policy identity/hash in the earlier plan contract.

**Outcomes preserved:** Policy repair coverage can grow across all families; provenance and deterministic selection remain. Large-estate configuration convenience is deferred, not required for initial correctness.

**Verify:** Missing entry declines; duplicate assignment errors; escaped pointer matches exactly; same operationId in two sources cannot cross-select; incomplete contact data that creates findings is rejected.

**Confidence:** High for the first providers. Broad deployment ergonomics need actual usage evidence.

## [P2] General scheduling and mutable repair states would add machinery to a synchronous batch

**Section / evidence:** Plan fields and execution sequence; [existing scheduler](../../motor/rule_runner.go), [transaction rollback](../../motor/auto_fix_transaction.go), [plugin registration](../../plugin/plugin_manager.go).

**Problem:** Edit dependencies, automatic plan merging, capability registration and a six-state outcome vocabulary could become a second scheduler, registry and mutable lifecycle. The required path is a synchronous plan, verification and write.

**Why it does not earn its complexity:** The inspected engine already owns rule execution and callback registration. Companion edits can be one ordered atomic plan. Independent plans need deterministic ordering; overlapping alternatives can be rejected without solving a dependency graph.

**Smaller design:** One direct built-in lookup in the existing rule/function ownership, with capability information derived from it. Preserve ordered plan groups, deduplicate identical edits and reject conflicts. Use separate immutable plan and verification results plus the writer's error/result; derive display status. Replace the old coordinator as the new path lands, retaining legacy signatures as adapters.

**Outcomes preserved:** Determinism, conflict visibility, plugin compatibility, rollback, machine output and reuse by all clients.

**Verify:** Repeated shared-source edits apply once; overlapping incompatible plans decline; operation order stays intact; CLI, plugin and external proposals use the same verifier. Fix-disabled lint does not construct edit graphs.

**Confidence:** Medium-high. Some metadata and lookups are necessary; the rejection is of independently maintained registries/state and a general scheduler, not the small dispatch table itself.

## [P2] Overlay export would add a translation path without a required initial consumer

**Section / evidence:** Edit format and client stage; [existing apply-overlay command](../../cmd/apply_overlay.go).

**Problem:** An additional export format requires translation and equivalence rules while the initial clients already accept a plan or repaired bytes.

**Why it does not earn its complexity:** Existing Overlay application remains available. The user requested built-in autofix and agent-friendly verification; neither requires a second export representation before shipping providers.

**Smaller design:** Defer generated Overlay export. Keep the existing command. Add export only when a consumer requires it, with exact supported-operation tests.

**Outcomes preserved:** OpenAPI transformations, portable repair plans, review previews and all intended initial clients.

**Verify:** CLI/library external-plan round trip and browser candidate-byte consumption do not depend on Overlay conversion.

**Confidence:** High for initial delivery. A future Overlay integration can justify the translation later.

## Crushed plan

- **Outcome and constraints:** Offer every built-in finding a repair or a clear disposition. Apply only eligible or explicitly selected changes; preserve format, source ownership, references, plugin contracts, Go N-1, exit codes, cancellation and bounded execution. Do not equate fewer findings with correctness.
- **Reuse:** Existing rule/functions, plugin callback registration, canonical node graphs, clone/rollback logic, lint/model rebuilding, error reporting and applicable serialization helpers. Keep writing outside the reusable planner.
- **Minimum sequence:** Fix current plugin/output defects; add identities and the small plan envelope for first providers; expose shared verification; ship guarded automatic cases behind opt-in; add the exact-scope policy file with real providers; extend provider coverage and client adapters. Use one coordinator throughout the final design.
- **Delete from the proposal:** Snapshot-index JSON Patch semantics, universal old-value tests, generic extension operations, a general dependency scheduler, separately maintained capability state, a mutable six-state repair lifecycle and claims that demand was measured.
- **Defer:** Policy globs/inheritance/remote loading, Overlay export, multi-file writes and schema rewrites whose equivalence is not proved. Root-only writes remain the initial boundary. The complete 157-rule matrix remains the scope record, not a promise to mutate every case.
- **Focused proof and completion:** Transaction-level no-op and rollback tests; valid JSON after insertion; standard patch parity; wrong-scope and suppression proposals rejected; final lint against frozen inputs; exact changed bytes and repair identities; idempotence; source-change/write-error handling. Measure lint, planning, repair plus verification and no-op passes on the benchmark corpus plus repair-heavy fixtures. Do not label the implementation complete from this document review.

## Review limits and document checks

Read-only source inspection and plan edits only. No production code, dependencies or WASM/site artifacts changed. The inventory checker still verifies the full rule snapshot. Document links and whitespace are checked. Existing probe/test results in the main plan remain prior audit evidence; they were not rerun as proof of this proposed design. Concurrent filesystem writers are not solved by hash-check-plus-rename: the revised plan explicitly states that limit rather than claiming compare-and-swap semantics.
