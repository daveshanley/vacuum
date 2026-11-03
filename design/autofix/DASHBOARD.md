# Autofix review in the existing dashboard

## Outcome and settled constraints

`vacuum lint --fix` repairs supported built-in and configured plugin findings without a second opt-in. `--fix --dry-run` performs the same candidate verification but writes no input, backup or output file. Remove the unreleased `--fix-builtins` and `--fix-plan` flags. Internal proposals remain library implementation data, never a required user document.

The terminal dashboard is the interactive review surface. Preserve its findings table, filtering, three-panel detail split, expanded code view and bottom command bar. Add behaviours in these areas; do not add a new layout or wizard. Review → select/skip/edit → preview → apply → remaining findings. Preview executes real verification in memory. Editing or a source reload invalidates preview. Apply writes exactly the reviewed candidate after checking the original source.

## Minimum design after plan-crush

1. Reuse the existing repair coordinator and atomic writer for CLI. Dry run shows verified edits and before/after counts while preserving machine-output contracts and exit codes. Do not build a second repair engine or persist a review session.
2. Add a small motor review owner tied to one immutable source snapshot and lint settings. Derive available fixes from existing providers and exact metadata policy rules. Selection is by finding identity, not by visible table row position. Regenerate proposals from selected findings; never delete arbitrary operations from a sequential patch.
3. Metadata values can be entered or changed for existing supported missing-field providers. Mechanical fixes support select/skip. Do not expose raw patch editing or invent metadata. Unsupported findings retain a reason. Group dependent edits and verify the whole candidate.
4. The dashboard uses the current detail area for value input and proposed action, the current code area/modal for a diff, and existing status/header areas for preview and completion counts. Context actions provide fixes filter, toggle, edit, preview, apply and cancel. Keep keyboard focus and escaping predictable.
5. Preview owns a candidate and final lint result. It shows selected/verified fixes, findings by severity before/after, and existing-score calculation where applicable. Applied statistics must distinguish repairs from diagnostics resolved. Never promise a perfect score. Failed verification retains user choices and leaves files unchanged.
6. Apply uses the shared source-checked writer. Source changes require a new review. Watch messages and async preview completions must not publish stale state. Cancel/quit releases resources and never writes. After apply, show remaining findings with the normal table and a completion summary.
7. CLI configuration remains the source of lint rules, ignores, reference access and timeouts. No separate configuration system. Dashboard repair is local root-file work; preserve existing external-source refusal limits. Saved reports and remote URLs remain read-only; comparison-filtered dashboards direct users to the full lint view for repair. Existing plugin callback changes remain verified in the CLI; do not imply arbitrary callbacks have editable metadata fields.

## Plan-crush findings addressed

- P1: Cutting operations out of a flattened plan can leave dependent patches invalid or attach the wrong findings. Regenerate selected providers from the original snapshot and verify as one transaction.
- P1: Editing after preview could write a different candidate than the user reviewed. Invalidate on every selection/value/source change and bind apply to exact original/candidate bytes.
- P2: A second dashboard or plan editor adds a layout and public format the user did not request. Reuse the existing panels, table, modal and key bar. Keep proposals internal.
- P2: Preview that only lists edits misstates successful repairs. Run actual application and relint in memory; show verified results and explicit failures.
- P2: New score logic would drift. Use existing severity/scoring semantics; retain findings that cannot be fixed.

## Verification and completion

- CLI: plain --fix repairs built-ins; plugin fixes still work; dry-run leaves source/output/backup unchanged; old flags absent; machine output parses; unchanged and failed verification cases are truthful.
- Motor: select a subset; edit two missing metadata values; skip a mechanical repair; preview final diagnostics and exact values; overlap/invalid values refused; no new diagnostics; repeat repair is idempotent; resource release.
- TUI: use real key messages for select/edit/cancel/preview/apply; preserve layout; edits invalidate preview; stale worker/watch results discarded; apply blocked without current preview; error keeps choices; final table/stats refresh.
- Command integration: real local source, same settings and writer; stale source leaves bytes untouched; backup/output semantics verified.
- Run an actual terminal dashboard session with a synthetic fixture, edit two values, preview, apply and inspect persisted bytes. Test dry run separately. Capture useful terminal evidence.
- Independent correctness, de-slop and draft PR adversarial reviews; fix confirmed findings; final full suite and hosted CI. No dependency upgrades or toolchain/WASM/site changes. The existing Bubbles text input may add its required transitive module checksums; do not build a custom text editor. No merge.
