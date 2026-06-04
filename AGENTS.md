# AGENTS.md

Operational guide for AI agents working in `github.com/daveshanley/vacuum`.

## TL;DR

- vacuum is a Go CLI and library for linting OpenAPI, AsyncAPI and JSON Schema documents, generating reports, bundling OpenAPI and JSON Schema documents where supported, running an LSP server, and generating OpenAPI docs through printing press.
- Main entry point: `vacuum.go` -> `cmd.Execute(...)` -> `cmd.GetRootCommand()`.
- The default OpenAPI and AsyncAPI user path is `vacuum lint <api-description>`. JSON Schema linting uses `vacuum schema <schema>` or `vacuum schema lint <schema>`.
- OpenAPI and AsyncAPI lint/report commands share helpers in `cmd/build_results.go`, `cmd/lint_shared.go`, and `motor/`. AsyncAPI context and default-ruleset wiring lives in `asyncapi/`, `cmd/asyncapi_ruleset.go`, `motor/asyncapi_applicator.go`, and `rulesets/asyncapi_rules.go`. JSON Schema command wiring lives in `cmd/schema*.go` and still executes rules through `motor/`.
- Go version is `1.25.0`. Node is only needed for the HTML report UI and npm package wrapper.
- The interactive HTML report is compiled only when UI assets exist and builds use `-tags html_report_ui`.
- Release/CI-shaped Go checks should usually run with `GOWORK=off` so local sibling checkouts do not hide committed module problems.
- Keep changes narrow. Do not rewrite generated assets, dependency locks, or docs unless the task requires it.

## Verify Changes

Run the smallest useful check first, then broaden when touching shared behavior.

```bash
gofmt -w <changed-go-files>
go test ./...
```

For official-build behavior, HTML report changes, or anything touching `html-report/`:

```bash
./scripts/build-ui-assets.sh
go test -tags html_report_ui ./...
make build
```

For release-shaped dependency, install, or CI validation:

```bash
GOWORK=off GOCACHE=/tmp/go-build go test ./...
GOWORK=off GOCACHE=/tmp/go-build go build ./...
```

For `html-report/ui` package-only changes:

```bash
cd html-report/ui
npm ci
npm run build
npm run lint
```

Before handoff for docs-only or config-only edits:

```bash
git diff --check -- <changed-files>
```

For JSON Schema command, ruleset, or schema Doctor changes, prefer focused checks before broader package runs:

```bash
go test ./cmd -run 'Schema'
go test ./motor -run 'JSONSchema'
go test ./jsonschema ./functions/jsonschema ./functions/schemachecks ./rulesets
```

For AsyncAPI linting, ruleset, or context changes, prefer focused checks before broader package runs:

```bash
go test ./cmd -run 'AsyncAPI|Lint|Report|Dashboard|GenerateRuleset'
go test ./motor -run 'AsyncAPI'
go test ./asyncapi ./functions/asyncapi ./rulesets
```

## Repo Map

```text
vacuum.go                    binary entry point and ldflags pass-through
cmd/                         Cobra commands, CLI flags, rendering, reports, docs and schema commands
asyncapi/                    AsyncAPI context, detection, and libasyncapi bridge helpers
motor/                       rule execution engine, document/index setup, result collection
model/                       rules, result models, reports, categories, test fixtures
rulesets/                    built-in rulesets, schemas, rule aliases, example rulesets
functions/core/              Spectral-compatible core rule functions
functions/asyncapi/          AsyncAPI-specific rule functions
functions/jsonschema/        JSON Schema rule functions and synthetic validation rules
functions/openapi/           OpenAPI-specific rule functions
functions/owasp/             OWASP rule functions
functions/schemachecks/      Shared schema sanity/type checks used by OpenAPI and JSON Schema rules
jsonschema/                  JSON Schema dialect, metaschema, Doctor, and reference helpers
plugin/javascript/           JavaScript custom-function runtime, event loop, fetch support
plugin/sample/               sample Go and JS custom functions
language-server/             OpenAPI LSP server integration
html-report/                 Go HTML report generator and embedded asset switch
html-report/ui/              TypeScript/Webpack UI bundle for HTML reports
tui/                         terminal UI, dashboard, tables, styles, markdown rendering
utils/                       change detection, ignore matching, path location, HTTP helpers
parser/                      JSON/YAML schema validation helpers
vacuum-report/               persisted report format and JUnit support
upgrade/                     release lookup, update notices, install-method detection, self-upgrade actions
npm-install/                 npm postinstall binary downloader for @quobix/vacuum
scripts/build-ui-assets.sh   builds HTML report UI assets for official builds
BUILD_PACKAGING.md           package-manager build and ldflags guidance
```

## Command Surfaces

Root command registration lives in `cmd/root.go`. Current subcommands include:

- `lint`
- `report`
- `spectral-report`
- `html-report`
- `dashboard`
- `docs`
- `generate-ruleset`
- `generate-ignorefile`
- `version`
- `language-server`
- `upgrade`
- `bundle`
- `schema`
- `apply-overlay`
- `open-collection`

When adding or changing flags, check every command surface that shares the behavior. For example, change filtering and resolved-reference behavior span `lint`, `report`, `spectral-report`, `html-report`, `dashboard`, and sometimes `language-server` or `docs`.

`vacuum schema` is the first-class JSON Schema surface:

- `vacuum schema <input...>` and `vacuum schema lint <input...>` lint JSON Schema documents.
- `vacuum schema bundle <input> [output]` bundles one schema entry document; use `--stdout` for piping.
- Schema inputs support explicit files, `--globbed-files`, folders, and exclusive `--stdin/-i`.
- Folder inputs default to recursive `.json`, `.yaml`, and `.yml`; `--include` and `--exclude` refine folder discovery. Explicit files and globs may use any filename or extension.
- Schema mode uses `json-schema-recommended` by default, supports custom rules/functions, and must not run OpenAPI-only rules.
- Schema bundling rewrites ordinary external `$ref` values into root `$defs`; dynamic/recursive references are preserved rather than resolved as ordinary refs.

`vacuum lint` is the first-class OpenAPI and AsyncAPI surface:

- OpenAPI and AsyncAPI documents use `vacuum lint <input...>` plus the `report`, `dashboard`, `html-report`, and `spectral-report` report surfaces.
- AsyncAPI 3 documents are auto-detected and use `asyncapi-recommended` by default, or the full AsyncAPI ruleset when hard mode is enabled.
- OpenAPI-only commands such as `bundle`, `docs`, `apply-overlay`, and `open-collection` must reject AsyncAPI clearly instead of falling through an OpenAPI path.

## Core Rules

- Prefer existing command helpers in `cmd/` over creating parallel execution paths.
- Keep shared lint behavior in `BuildResults*`, `LintLoadedSpec`, `motor.RuleSetExecution`, or `motor.ExecutionOptions` when multiple commands need it.
- Preserve the exit-code contract: `0` clean, `1` violations at or above threshold, `2` parse/input/tool error.
- Use `go.yaml.in/yaml/v4` where current code does; do not casually mix YAML libraries.
- Keep rule IDs, categories, and built-in rule constants centralized in `rulesets/` and `model/`.
- For result paths, preserve both `Path` and `Paths` semantics. Many report and diff workflows depend on stable path output.
- For AsyncAPI, preserve `RuleFunctionContext.AsyncAPI`, libasyncapi diagnostics, node-path mapping, and the default ruleset selection path. Reports, stats, snippets, and diagnostics should remain first-class outputs.
- For JSON Schema, preserve the schema-only Doctor path, libopenapi index/rolodex reference behavior, and low-level YAML/JSON node fidelity. Line/column, snippets, JSONPath output, and `$ref` diagnostics depend on that stack.
- Do not remove or weaken panic recovery, timeouts, lookup timeouts, or circular-reference controls without focused tests.
- For custom JS functions, keep fetch security defaults intact: HTTPS only unless `--allow-http`, and private network access only with `--allow-private-networks`.

## HTML Report UI

- `html-report/assets_stub.go` is used when the `html_report_ui` tag is absent.
- `html-report/assets_html_report_ui.go` embeds `html-report/ui/build/static/js/vacuumReport.js` and `hydrate.js` when the tag is present.
- `./scripts/build-ui-assets.sh` runs `npm ci` and `npm run build` in `html-report/ui`, using `.cache/npm` unless `NPM_CACHE_DIR` is set.
- The script removes `html-report/ui/node_modules` unless `KEEP_NODE_MODULES=1`.
- Official releases and `make build` build with `-tags html_report_ui`.
- Do not edit built files under `html-report/ui/build/`; edit `html-report/ui/src/` and regenerate.

## Rules And Functions

- Built-in functions implement `model.RuleFunction`.
- A function must provide `RunRule`, `GetSchema`, and `GetCategory`.
- Tests usually live beside the function they cover: `functions/core`, `functions/openapi`, `functions/asyncapi`, `functions/jsonschema`, `functions/schemachecks`, or `functions/owasp`.
- Add new built-in rule IDs and docs metadata through the existing `rulesets` patterns.
- Example rulesets belong under `rulesets/examples/`; schema changes belong under `rulesets/schemas/`.
- AsyncAPI-specific built-ins should use `functions/asyncapi`; shared core functions are fine when the rule behavior is genuinely format-neutral.
- JSON Schema-specific built-ins should use `functions/jsonschema`; shared schema semantics belong in `functions/schemachecks` so OpenAPI and JSON Schema do not drift.
- Custom Go plugin behavior is demonstrated in `plugin/sample/`.
- Custom JavaScript runtime behavior lives under `plugin/javascript/`; async, Promise, and fetch behavior should be tested there.

## Docs Command

- `cmd/docs.go` is the CLI entry point for generated API docs.
- `cmd/docs_config.go` maps printing press config file values into CLI options.
- `cmd/docs_diagnostics.go` runs vacuum lint diagnostics and wires results into docs output.
- `cmd/docs_press.go` calls the printing press engine from `github.com/pb33f/doctor`.
- Single-spec and aggregate/catalog paths are both first-class; preserve both when changing docs behavior.
- Defaults matter: output is `./api-docs`, port is `9090`, and diagnostics are enabled unless `--no-diagnostics`.

## Upgrade And Update Checks

- `cmd/upgrade.go` is the CLI entry point for `vacuum upgrade`.
- `upgrade/` owns GitHub release lookup, update-check cache state, install context detection, notice rendering, and the npm/Homebrew/shell upgrade actions.
- `cmd/update_check.go` starts the non-blocking update check and flushes notices around command execution.
- Preserve machine-output behavior: update notices must not interfere with `--stdout`, `--silent`, `--pipeline-output`, CI, or non-terminal output.
- `--no-update-check` disables the update check for a single run and should remain available as a persistent root flag.

## Config And Paths

- Config file name is `vacuum.conf.yaml`.
- Default lookup order is current working directory, then `$XDG_CONFIG_HOME` if set, otherwise `$HOME/.config`.
- `--config` may point at an explicit config file.
- Environment variables use `VACUUM_<FLAG>` with dashes replaced by underscores.
- Relative paths from config should resolve against the config file directory when appropriate; see `ResolveConfigPath`.
- `go.work` is ignored and may exist locally. Use `GOWORK=off` for dependency and release behavior.

## Generated And Local Artifacts

Do not treat these as source unless the task explicitly says so:

- `bin/`
- `dist/`
- `.cache/`
- `html-report/ui/build/`
- `html-report/ui/node_modules/`
- `node_modules/`
- `api-docs/`
- `_speakeasy_openapi/`
- `eden/`
- `speakeasy-refs/`
- `vacuum-report-*.json`
- temporary reports produced under `cmd/`

Some of these may be ignored by local `.git/info/exclude` rather than `.gitignore`.
Check `git status --short` before editing. Preserve unrelated user changes.

## Dependency Work

- Main module dependencies live in `go.mod` and `go.sum`.
- npm wrapper dependencies live in root `package.json` and `package-lock.json`.
- HTML report UI dependencies live in `html-report/ui/package.json` and `html-report/ui/package-lock.json`.
- For Go dependency bumps, prefer the narrow requested bump, then verify with `GOWORK=off`.
- Avoid `go mod tidy` churn in a dirty main worktree. If tidy behavior matters, use a clean temporary worktree or confirm the diff is only `go.mod`/`go.sum`.
- The npm package version is bumped during publish; root `package.json` may show `0.0.0` in source.
- `npm pack --dry-run` is useful for checking npm package contents.

## Release And Packaging

- GoReleaser config is `.goreleaser.yaml`.
- Release builds run `./scripts/build-ui-assets.sh` before compiling.
- Release binaries include HTML report UI assets with `-tags html_report_ui`.
- Docker builds use Node 20 for UI assets and Go 1.25 for the binary.
- npm publish uses trusted publishing in `.github/workflows/publish.yaml`.
- `go install github.com/daveshanley/vacuum@<version>` does not include HTML report UI bundles. This is expected.
- Do not add `replace` directives to released module state; they break `go install module@version` and `go run module@version`.

## Testing Guidance

- Command behavior: add focused tests under `cmd/`.
- Rule execution behavior: add or update tests under `motor/`.
- Built-in rule functions: test in the relevant `functions/...` package.
- AsyncAPI command, context, and default-ruleset behavior: test under `cmd/`, `asyncapi/`, `motor/`, `functions/asyncapi/`, and `rulesets/`.
- JSON Schema command behavior: test under `cmd/`; dialect, metaschema, Doctor, and reference helpers: test under `jsonschema/`; schema-only motor behavior: test under `motor/`.
- Ruleset parsing/aliases/default rules: test under `rulesets/`.
- Language server behavior: test under `language-server/`.
- HTML report Go behavior: test under `html-report/`; use `-tags html_report_ui` for embedded UI behavior.
- Use existing fixtures in `model/test_files`, `cmd/test_data`, `motor/test_data`, and `parser/schemas/test_files` before adding new ones.
- For issue repros, keep fixtures tiny and name tests after the issue or behavior.

## Git Hygiene

- Never revert user changes unless explicitly asked.
- Keep generated or vendored noise out of commits.
- Run `git diff --check` before handoff.
- When a command fails because of sandbox DNS, cache, or socket restrictions, do not call it a product regression. Rerun with temp caches or the required permissions.
- Prefer exact, reproducible commands in final notes so the next agent can rerun the same proof.
