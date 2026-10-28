# Report output formats

Use `vacuum lint --format` to export lint findings. Without `--format`, lint keeps its existing terminal output.

```sh
vacuum lint api.yaml --format sarif --output vacuum.sarif
vacuum lint api.yaml --format json > findings.json
vacuum lint api.yaml --format github-actions
vacuum lint api.yaml --format json --format junit --output json=findings.json --output junit=junit.xml
vacuum lint --globbed-files 'specs/*.yaml' --format gitlab --output gl-code-quality-report.json
```

The supported names match OpenLint 1.0.0:

| Format | Output |
| --- | --- |
| `json` | Spectral-style diagnostic array with zero-based ranges and typed JSONPath segments |
| `stylish` | Compact terminal table, grouped by source |
| `junit` | JUnit XML; the lint failure threshold controls failure elements |
| `html` | Standalone HTML table containing every finding, with no external assets or scripts |
| `text` | One finding per line with source, line, column, severity, rule, and message |
| `teamcity` | Escaped TeamCity inspection service messages |
| `pretty` | Spaced terminal table, grouped by source |
| `github-actions` | GitHub workflow annotations using the existing annotation renderer |
| `sarif` | SARIF 2.1.0 with rule metadata, source locations, and severity levels |
| `code-climate` | Code Climate issue array with stable fingerprints and one-based lines |
| `gitlab` | Alias for `code-climate`, for GitLab Code Quality reports |
| `markdown` | Escaped finding table with source locations |

These are format-compatible exports of vacuum findings. They do not copy OpenLint's rule behavior, branding, terminal styling, or formatting defects. For example, Code Climate line numbers are one-based.

Multiple formats share one lint run. Use `--output format=path` for named destinations. At most one format can use stdout. Use `-` explicitly for stdout. A single format also accepts `--output path`. Output filenames must be distinct and must not overwrite the input document, loaded OpenAPI reference documents, or the `--fix-file` destination. File reports are replaced only after rendering succeeds. A later destination can still fail after an earlier report has been written.

`--format` suppresses banners, progress, update notices, and debug logs. It cannot be combined with the existing terminal presentation switches such as `--details`, `--silent`, `--pipeline-output`, or `--github-annotations`. Use `--format github-actions` for annotation exports. `--errors` filters displayed findings; `--category` selects findings before the failure outcome is computed. Existing lint severity and score thresholds still apply. Exit codes remain 0 for success, 1 for violations, and 2 for input/tool errors, including report write errors. Errors go to stderr.

Existing GitHub annotations now count Unicode code points instead of UTF-8 bytes when computing scalar end columns. This corrects the end position for non-ASCII text.

The existing `report`, `spectral-report`, `html-report`, `--pipeline-output`, and `--github-annotations` commands and switches retain their contracts. The existing interactive `html-report` remains available for charts, navigation, and snippets. The new `--format html` is a complete portable table, available even without the `html_report_ui` build tag. JSON Schema's existing `schema --format json` output is unchanged.

In CI, run from the repository root so SARIF and Code Quality paths are repository-relative. Remote URLs retain their source identity; GitHub annotations omit URL file fields. Code Quality consumers need local files in the checkout to attach findings. JSON paths preserve literal keys such as `a.b` and array indexes as numbers. Human-readable reports use one-based locations.

## Implementation reuse

- The command uses the existing lint execution, ignore filters, result collection, and exit policy.
- Location conversion uses the existing annotation span cache and report range types.
- GitHub workflow escaping and serialization are shared with the existing annotation command.
- JUnit exports reuse the existing XML model types; a separate adapter supports every lint failure threshold and uncategorized findings.
- Markdown exports use the compact writer alongside `tui.RenderMarkdownTable`.
- The lightweight HTML template lives in `html-report` and uses Go's autoescaping `html/template`.

There is no raster step. All supported formats are text, structured data, or HTML. Exporting them does not require a browser, Node, or an image renderer.
