# Lint Arazzo workflows

Vacuum detects Arazzo 1.0 and 1.1 YAML or JSON documents and selects
`arazzo-recommended`. Use the normal lint and report commands:

```shell
vacuum lint workflows.yaml
vacuum lint workflows.yaml --details --snippets
vacuum spectral-report workflows.yaml --stdout
vacuum report workflows.yaml --stdout
vacuum html-report workflows.yaml workflows.html
cat workflows.yaml | vacuum spectral-report --stdin --stdout --base ./specs
```

The dashboard, ignore files, inline `x-lint-ignore`, custom core/JavaScript/Go
rules, severity thresholds and language server use the same rule execution path.
HTML reports require a build with the report UI, as for OpenAPI and AsyncAPI.
This feature validates workflows; it does not execute them. Bundling and docs
generation for Arazzo are outside its scope.

## Validation and source documents

`libopenapi-validator` v0.16.0 checks structure, identifiers, references,
parameters, expressions, selectors, dependency cycles, input schemas and source
types. Validation runs once per lint execution. Its findings become normal
Vacuum rules with source locations and JSONPaths.

Source descriptions can refer to OpenAPI, AsyncAPI or other Arazzo documents.
The CLI enables file and HTTP(S) lookups by default, as it does for other API
documents. Relative sources resolve from the workflow file; `--base` overrides
that location. Existing client certificate, CA and TLS flags also apply.
`--remote=false` disables remote lookup. An explicit or inferred local base
still allows local files, consistent with Vacuum's existing reference policy.
Library callers control lookup with `RuleSetExecution.AllowLookup` and `Base`.
Only HTTP(S) and local file sources are supported.

Source reads are bounded. The validator enforces its node, depth, total source,
byte and diagnostic limits. Source requests honor execution cancellation and
the rule timeout. No workflow requests, actions or scripts are executed.

When a check cannot run, `arazzo-validation-incomplete` reports a warning at the
affected location. Examples include disabled source lookup, XPath, legacy
JSONPath, runtime-dependent conditions and unsupported external input-schema
references. A warning does not assert that the document is invalid. Use
`--fail-severity warn` if incomplete checks must fail your lint job. A failed
enabled lookup, parse failure, unsupported version or timeout is a tool error.
The lint exit codes remain `0` for no findings at the selected threshold, `1`
for findings at that threshold, and `2` for input or tool errors.

External source findings retain their source in reports. In the editor, they
appear on the workflow's source declarations and link to the actual source
line through related diagnostic information. Source findings use the inline
ignore directives in their own document. `--original` rechecks both source
graphs, so a changed API can produce a new finding even when the workflow
document has not changed.

## Rules and configuration

Use `vacuum:arazzo` or `arazzo-recommended` in `extends`. `spectral:arazzo` is an
alias for Vacuum's recommended Arazzo rules. `--hard-mode` enables the full
Arazzo ruleset, including optional authoring advice; it does not enable OpenAPI
rules for an Arazzo document.

```yaml
extends: [vacuum:arazzo]
rules:
  arazzo-validation-incomplete: error
  arazzo-info-summary: off
  workflow-title:
    formats: [arazzo]
    given: $.info
    then:
      field: title
      function: truthy
```

Formats are `arazzo` (both versions), `arazzo1_0`, and `arazzo1_1`.

| Rule | Check | Default severity |
| --- | --- | --- |
| `arazzo-structure` | Document structure | Error |
| `arazzo-duplicate-id` | Duplicate identifiers | Error |
| `arazzo-reference` | Workflow, step and source targets | Error |
| `arazzo-parameter` | Parameters and linked operation metadata | Error |
| `arazzo-expression` | Runtime expressions | Error |
| `arazzo-selector` | Selectors and success criteria | Error |
| `arazzo-dependency` | Prerequisites and cycles | Error |
| `arazzo-input-schema` | Input schemas | Error |
| `arazzo-source-type` | Source document types | Error |
| `arazzo-validation-incomplete` | Checks that could not finish | Warning |
| `arazzo-info-description` | Info description | Warning |
| `arazzo-workflow-description` | Workflow descriptions | Warning |
| `arazzo-step-description` | Step descriptions | Warning |
| `arazzo-no-script-tags-in-markdown` | Script tags in titles/descriptions | Warning |
| `arazzo-info-summary` | Info summary | Hint |
| `arazzo-workflow-summary` | Workflow summaries | Hint |
| `arazzo-advisory` | Identifier and forward-reference advice | Warning, hard mode |
| `arazzo-step-operationPath` | Prefer operation identifiers | Hint, hard mode |

Generate an editable ruleset with `vacuum generate-ruleset arazzo-recommended`
or `vacuum generate-ruleset arazzo-all`.

## Spectral migration

The `spectral:arazzo` alias selects the native validator and rules above.
It does not run Spectral's JavaScript rule functions or promise identical
messages and rule IDs. Move semantic rule overrides to the native diagnostic
IDs: for example, workflow/step uniqueness checks use `arazzo-duplicate-id`,
and prerequisite checks use `arazzo-dependency`. Description and summary rule
IDs are retained. Test existing custom rules and ignore entries against a
generated report when migrating.

## Library usage

```go
result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
    Spec:         workflowBytes,
    SpecFileName: "specs/workflows.yaml",
    AllowLookup:  true,
    RuleSet:      rulesets.BuildDefaultRuleSets().GenerateArazzoRecommendedRuleSet(),
})
defer result.ReleaseOwnedResources()
// Handle result.Errors before reading result.Results.
// result.Arazzo.Validation contains validator coverage when validation rules ran.
```

Use `ApplyRulesToRuleSetWithOptions` to provide a caller context, a total run
timeout, or a rule concurrency limit. Custom-only rulesets run without invoking
the validator or loading source descriptions.
