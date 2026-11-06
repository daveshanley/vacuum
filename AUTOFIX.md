# Autofix

`vacuum lint api.yaml --fix` applies supported built-in repairs and configured
plugin fixes. A rule must be enabled in the active ruleset before it can supply
a repair. Automatic repairs do not invent descriptions, contact details or
other values that require the API author's decision.

```sh
# Verify and inspect changes without writing files.
vacuum lint api.yaml --fix --dry-run

# Write a separate output file.
vacuum lint api.yaml --fix --fix-file api.fixed.yaml

# Apply repairs to a JSON Schema document.
vacuum schema schema.json --fix
```

Dry run performs real application and verification in memory. Its report shows
the candidate's remaining findings. Human-readable output includes a bounded
diff; omitted portions are marked. Structured reports retain their usual
format. No source, output or backup file is written by dry run.

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
| `e` | Open the value editor modal |
| `v` | Apply selected edits in memory and verify the candidate |
| `x` | Open the code view or preview diff |
| `a` | Write the current verified candidate |
| Escape | Cancel an edit, close a modal, or cancel review |

Descriptions, summaries and operation IDs accept plain text. Contact and
license values accept JSON objects with supported string fields. Mechanical
repairs support selection, not arbitrary patch editing.

The fix modal shows the selected issue above two source panes: Currently and
Autofixed. Both use the dashboard's syntax highlighting. Removed or changed
lines are blue on the left; added or changed lines are yellow on the right.
Unchanged lines keep their normal syntax colors. Solid pink borders frame the
comparison, with one pipe-separated control row. The proposal uses formatted
excerpts of the selected source location. It is not yet
verified and does not write a file. Use `v` to verify the selected batch; the
preview diff shows the actual candidate edits, with omitted sections marked.

Supported editable fixes open with proposed content already inserted in the
right-hand source pane. The dashboard drafts descriptions, summaries and unique
operation IDs from source context. Contact proposals contain a name, without
invented email addresses or URLs. License proposals use `Unspecified`, without
asserting a license grant. Generated values are labelled and start unselected.
Press Enter to select the displayed proposal, or `e` to change its prefilled
value. Review generated wording and metadata before applying. These dashboard
suggestions do not change CLI `--fix` behavior. Verification rejects a proposal
if the active rules require more information or it introduces another finding.

Editing uses a separate modal. Enter saves the value for review; Escape keeps
the prior value. Cursor blinking does not resize the surrounding content.

The preview header shows verified repairs, severity counts and the existing
quality score before and after. Apply refreshes the table with remaining
findings and reports actual results. A successful repair does not guarantee a
score of 100: other findings may remain. Editing a selection or value invalidates
the preview. Source reloads also discard the previous review.

Saved reports and remote URLs are read-only. Open the local specification to
apply changes. Comparison views using `--original` or `--changes` direct users
to the full lint view for repairs. Dashboard review covers built-in mechanical
and metadata repairs; configured plugin callbacks run through CLI `--fix`.

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
or changing inline ignore directives. Verification uses the active rules,
severities, ignores and reference settings. Relinting does not prove that an
arbitrary API contract change preserves its intended meaning; built-in repairs
therefore have narrow eligibility checks. If candidate verification fails, the
CLI keeps the baseline report, writes no candidate bytes, and explains the
declined repair on standard error (unless `--silent` is set).

The CLI checks original source bytes before writing, stages the candidate in
the destination directory and renames it. In-place writes create a temporary
backup. Symlink inputs and output paths that alias the input through a hardlink
are rejected. Replacing one path leaves other hardlinks on their original
inode. The source check and rename are not a filesystem compare-and-swap.

Only the root file is written. OpenAPI verification reuses captured external
reference bytes. AsyncAPI/Arazzo documents with external sources are refused
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
