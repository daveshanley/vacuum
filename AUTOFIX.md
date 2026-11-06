# Autofix

`vacuum lint api.yaml --fix` proposes supported built-in repairs. Every fix
requires review before it is written. A rule must be enabled in the active
ruleset before it can supply a proposal. Supported cases include structural
repairs, generated metadata, and schema-derived examples. A rule can have
supported and unsupported cases; the rule ID alone does not guarantee a fix.

```sh
# Verify and inspect changes without writing files.
vacuum lint api.yaml --fix --dry-run

# Review fixes, then write a separate output file after acceptance.
vacuum lint api.yaml --fix --fix-file api.fixed.yaml

# Review repairs to a JSON Schema document.
vacuum schema schema.json --fix
```

Dry run performs real application and verification in memory. Its report shows
the candidate's remaining findings. Human-readable output includes a bounded
diff; omitted portions are marked. Structured reports retain their usual
format. No source, output or backup file is written by dry run.

## Review in the console

The console lists available fixes with numbers. Enter numbers separated by
spaces, or press Enter to review all available fixes. Esc cancels.
Vacuum verifies the selection, then shows contextual diff snippets and the
contract impact, including any breaking changes or unknown impact.

Press Enter to apply the reviewed changes. Press Esc to cancel without writing.
Use the dashboard to edit values or
inspect individual proposals. A conflicting selection is declined in full;
Vacuum does not silently remove selected fixes to make the batch pass.

Noninteractive `--fix` cannot write. Use `--fix --dry-run` in automation to
inspect projected results. Dry run attempts all available proposals and reports
conflicts or validation failures without writing. Review text and decline
reasons use standard error, so structured reports on standard output retain
their format. `--silent` suppresses dry-run review output and cannot accept
changes through an unseen prompt. The source is checked again before an accepted
candidate is written.

## Review in the dashboard

Run `vacuum dashboard api.yaml`. The existing findings table, detail panels,
code view and command bar provide the review controls. A brief fix-available
indicator calls attention to supported repairs after discovery finishes:

| Key | Action |
| --- | --- |
| `f` | Open review and filter supported fixes |
| Space | Select or skip a fix |
| Enter | Open before/after; select or skip the displayed proposal; save a value in the editor |
| ↑ / ↓ | Move to the previous or next fix while the comparison stays open |
| Page Up / Page Down | Scroll the current comparison |
| ← / → | Pan long source lines horizontally |
| `e` | Open the value editor modal |
| `v` | Verify the displayed proposal, or selected fixes from the list, and open the change view |
| `x` | Open the code view or expand the verified preview |
| `[` / `]` | Move between changes in a verified preview |
| `t` | Switch between the change view and complete file diff |
| `a` | Open the save modal: choose the destination and bundling, review the checked output, then save |
| Escape | Cancel an edit, close a modal, or cancel review |

Descriptions, summaries and operation IDs accept plain text. Contact and
license values accept JSON objects with supported string fields. Mechanical
repairs support selection, not arbitrary patch editing.

The fix modal shows the selected issue above two source panes: Currently and
Autofixed. Both use the dashboard's syntax highlighting. Removed or changed
lines are blue on the left; added or changed lines are yellow on the right.
Unchanged lines keep their normal syntax colors. YAML and JSON source lines do
not wrap. Crop markers show hidden text; use Left and Right to pan both panes.
The verified save diff also scrolls horizontally instead of wrapping source lines. Solid pink borders frame the
comparison, with one pipe-separated control row. The proposal uses formatted
excerpts of the selected source location. It is not yet
verified and does not write a file. Use `v` for a preview without saving, or press `a` to start the save flow.
In Currently / Autofixed, `v` verifies only the displayed proposal, including
your edited value. It does not select that fix for saving. From the fix list,
`v` verifies all selected fixes. Pressing `v` in an existing verified view keeps
the current change and scroll position. The `a` save flow always checks the full
selected batch with its output options.

Verified previews use the same change renderer as OpenAPI Changes. Each card
shows the action, value change, contract impact, and source context. Large values
and source blocks use marked excerpts. Press `t` to inspect the complete file
diff, including formatting and bundling changes. If semantic changes are not
available, Vacuum opens the complete file diff. External source locations are
not highlighted against the root file.

Supported editable fixes open with proposed content already inserted in the
right-hand source pane. The dashboard drafts descriptions, summaries and unique
operation IDs from source context. Contact proposals contain a name, without
invented email addresses or URLs. License proposals use `Unspecified`, without
asserting a license grant. Generated values are labelled. All fixes start unselected.
Press Enter to select the displayed proposal, or `e` to change its prefilled
value. Review generated wording and metadata before applying. The console uses
the same proposals. Verification rejects a proposal if it introduces a new integrity defect or lint finding, or leaves its target unresolved.
An invalid example can be replaced with a validated instance generated from its
schema. Existing valid data is preferred where supported. Generation is bounded
and deterministic; unsupported constraints or failed validation cause a decline.
Example generation does not authorize replacing a schema default.

Editing uses a separate modal. Enter saves the value for review; Escape keeps
the prior value. Cursor blinking does not resize the surrounding content.

The save modal asks you to update the current specification or create a new
file. For a new file, enter its name; relative names resolve beside the source.
Use `.yaml`, `.yml`, or `.json` to select the output format. Next, choose whether
to keep references, inline external references, or compose reusable components.
Bundling uses the native OpenAPI 3 bundler. Other document formats must keep
references. A copy in another directory is declined if the final output still
contains location-dependent references.

Vacuum verifies the exact output after format conversion and bundling. The modal
shows the destination, change view, contract impact, fix counts and score.
Press `t` for the complete file diff. Press
Enter to save. Breaking or unknown impact needs a second Enter to accept that
impact. Press `e` to change the output options, or Escape to cancel. No earlier
`v` preview is required. Errors stay in the modal; failed verification cannot be
overridden.

Updating the source refreshes the findings table. Creating a copy leaves the
source and its dashboard findings unchanged. Existing output files cannot be
replaced. The completion modal shows the saved path, remaining findings and
score. A successful repair does not guarantee a
score of 100: other findings may remain. Editing a selection or value invalidates
the preview. Source reloads also discard the previous review.

Saved reports and remote URLs are read-only. Open the local specification to
apply changes. Comparison views using `--original` or `--changes` direct users
to the full lint view for repairs. Dashboard and console review cover supported
built-in proposals. CLI `--fix`
currently declines configured custom fix callbacks, including in dry-run mode.
The library callback APIs retain their existing behavior; they do not provide
CLI review or acceptance.

## Supply exact metadata values

A local policy supplies values for missing contact, license, description,
summary and operation ID fields. Values apply only to the named source and
JSON Pointer. They do not replace existing empty, null or conflicting values.

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

The optional catalog maps an exact source and unique operation ID to a value:

```yaml
api.yaml:
  listPets: List the pets visible to the authenticated caller.
```

```sh
vacuum lint api.yaml --fix --fix-policy policy.yaml
```

Source and catalog paths resolve relative to the policy file. The rule must be
enabled: `info-contact`, for example, needs an explicit ruleset setting or
`--hard-mode` with the OpenAPI defaults. Missing catalog entries, ambiguous
operation IDs and overlapping assignments are refused. Unknown fields,
wildcard scopes and remote catalogs are not accepted.

## Verification and file handling

Repairs must resolve their targeted findings without introducing new diagnostics
or changing inline ignore directives. An internal integrity check compares
baseline and candidate defects independently of the selected lint rules and
ignores. It checks structure, references and identifiers through the existing
format validators. Existing unrelated defects can remain; new defects block
application. The selected lint rules also verify the repair targets.

For supported OpenAPI documents, libopenapi compares the original contract with
the complete selected candidate. A breaking result includes the detected changes.
An unsupported format, incomplete comparison, or comparison error produces
unknown impact. No breaking changes detected is not a guarantee of runtime
compatibility. The current native comparator can report breaking changes for
description-only edits in union schemas. Review the source diff when assessing
that warning; Vacuum does not suppress the native result.
User acceptance of breaking or unknown impact cannot override a
failed integrity check. Changing a value or selection invalidates verification
and the impact result. If candidate verification fails, the CLI keeps the baseline report, writes no candidate bytes, and explains the
declined repair on standard error (unless `--silent` is set).

The CLI checks original source bytes before writing, stages the candidate in
the destination directory and renames it. In-place writes create a temporary
backup. Symlink inputs and output paths that alias the input through a hardlink
are rejected. Replacing one path leaves other hardlinks on their original
inode. The source check and rename are not a filesystem compare-and-swap. Dashboard
copies use atomic create-only publication from the staged file, so a destination
created during review or save cannot be overwritten.

Only the root file is written. OpenAPI verification reuses captured external
reference bytes. Arazzo verification also reuses captured source bytes and redirect
aliases; it does not read files or fetch URLs again. AsyncAPI documents with external sources are refused
for repair until equivalent frozen-source verification is available. LSP
repairs return versioned editor edits and do not write files directly.

Multi-document YAML, anchored or aliased trees, ambiguous keys and edits that
would lose comments are declined where the repair cannot preserve them.
Reference-bearing documents have conservative restrictions on array removal
and reordering. Parsing is limited to 64 MiB and depth 256. A repair batch has
at most 10,000 patch operations and a cumulative copied-node budget of 256 MiB.
These limits are not a bound on total process memory.

For custom Go callback setup and its in-place mutation contract, see
[Auto-fixing rule violations](README.md#auto-fixing-rule-violations).
