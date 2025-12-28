![logo](logo.png)

# vacuum - The world's fastest OpenAPI & Swagger linter.

![build](https://github.com/daveshanley/vacuum/workflows/Build/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/daveshanley/vacuum)](https://goreportcard.com/report/github.com/daveshanley/vacuum)
[![discord](https://img.shields.io/discord/923258363540815912)](https://discord.gg/UAcUF78MQN)
[![Docs](https://img.shields.io/badge/godoc-reference-5fafd7)](https:/-/pkg.go.dev/github.com/daveshanley/vacuum)
[![npm](https://img.shields.io/npm/dm/@quobix/vacuum?style=flat-square&label=npm%20downloads)](https://www.npmjs.com/package/@quobix/vacuum)
[![Docker Pulls](https://img.shields.io/docker/pulls/dshanley/vacuum?style=flat-square)](https://hub.docker.com/r/dshanley/vacuum)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)

An **ultra-super-fast**, lightweight OpenAPI linter and quality checking tool, written in golang and inspired by [Spectral](https://github.com/stoplightio/spectral).

It's **fully compatible** with existing [Spectral](https://github.com/stoplightio/spectral) rulesets.

## Install using [homebrew](https://brew.sh) tap

```
brew install daveshanley/vacuum/vacuum
```

## Install using [npm](https://npmjs.com)

```
npm i -g @quobix/vacuum
```

## Install using [yarn](https://yarnpkg.com/)

```
yarn global add @quobix/vacuum
```

## Install using curl

```bash
curl -fsSL https://quobix.com/scripts/install_vacuum.sh | sh
```

### For CI/CD environments 

To avoid GitHub API rate limiting in automated environments, set a GitHub token:

```bash
# Using repository token (GitHub Actions)
GITHUB_TOKEN=${{ secrets.GITHUB_TOKEN }} curl -fsSL https://quobix.com/scripts/install_vacuum.sh | sh

# Using personal access token
GITHUB_TOKEN=your_github_token curl -fsSL https://quobix.com/scripts/install_vacuum.sh | sh
```

#### GitHub Actions example

```yaml
- name: Install vacuum
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}  # Increases rate limit from 60 to 5000 requests/hour
  run: |
    curl -fsSL https://quobix.com/scripts/install_vacuum.sh | sh
```

> **Note**: The GitHub token prevents intermittent installation failures in CI/CD environments caused by API rate limiting. 
> No additional permissions are required, the token only accesses public repository information.

## Install using [Docker](https://hub.docker.com/r/dshanley/vacuum)

The image is available at: https://hub.docker.com/r/dshanley/vacuum

```
docker pull dshanley/vacuum
```

> **Multi-platform support**: Docker images are available for both `linux/amd64` and `linux/arm64` architectures, including native ARM64 support for Apple Silicon Macs.

To run, mount the current working dir to the container and use a relative path to your spec, like so

```
docker run --rm -v $PWD:/work:ro dshanley/vacuum lint <your-openapi-spec.yaml>
```
Alternatively, you can pull it from
[Github packages](https://github.com/daveshanley/vacuum/pkgs/container/vacuum).
To do that, replace `dshanley/vacuum` with `ghcr.io/daveshanley/vacuum` in the above commands.

## Run with Go

If you have go >= 1.16 installed, you can use `go run` to build and run it:

```
go run github.com/daveshanley/vacuum@latest lint <your-openapi-spec.yaml>
```

---

## Sponsors
If your company is using `vacuum`, please considering [supporting this project](https://github.com/sponsors/daveshanley),
like our _very kind_ sponsors, past and present:


<a href="https://speakeasyapi.dev/?utm_source=vacuum+repo&utm_medium=github+sponsorship">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/sponsors/speakeasy-github-sponsor-dark.svg">
  <img alt="speakeasy'" src=".github/sponsors/speakeasy-github-sponsor-light.svg">
</picture>
</a>

[Speakeasy](https://speakeasyapi.dev/?utm_source=vacuum+repo&utm_medium=github+sponsorship)

<a href="https://scalar.com">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/sponsors/scalar-dark.png">
  <img alt="scalar" src=".github/sponsors/scalar-light.png">
</picture>
</a>

[scalar](https://scalar.com)

<a href="https://apideck.com">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/sponsors/apideck-dark.png">
  <img alt="apideck'" src=".github/sponsors/apideck-light.png">
</picture>
</a>

[apideck](https://apideck.com)

---

## Come chat with us

Need help? Have a question? Want to share your work? [Join our discord](https://discord.gg/UAcUF78MQN) and
come say hi!

## Documentation

🔥 **New in** `v0.22` 🔥: **Async Functions / Promises, Fetch & Batch mode in Custom JS Functions**

Do you want to call remote APIs in your vacuum javascript functions? What about async processing or the ability to use Promises?

vacuum now has its own event loop and will happily support `async` and `await`. Combined with a full implementation of [Fetch](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API).

Also added **Batch Mode**. This allows custom functions to receive the entire list of nodes, instead of firing the function 
for each result, so you can send all your data off to an API or an LLM, and have the ability parse and process everything at once
vs individually. 

- [Read all about async JS functions in vacuum](https://quobix.com/vacuum/api/custom-javascript-functions/#async-functions--promises)
- [Learn about using fetch in vacuum](https://quobix.com/vacuum/api/custom-javascript-functions/#fetch-api)
- [find out how batch mode works](https://quobix.com/vacuum/api/custom-javascript-functions/#batch-mode-processing)

---

`v0.21`: **Change detection filtering**

Want to see linting results on **just** the changes you have made to an OpenAPI document? Or want to see just the results on the differences between two documents? 
comes with a **what changed** mode. Using the new `--original` and `--changes-summary` global flags, you can filter out all the noise. 

[documentation for change detection](https://quobix.com/vacuum/commands/change-detection/) is available to learn more.

vacuum now supports [JSON Path Plus](https://github.com/JSONPath-Plus/JSONPath) annotations. This means that vacuum is 
compliant with **All Spectral Paths**. One of the last remaining gaps between vacuum and spectral has been closed. 

[See all the newly supported annotations](https://quobix.com/vacuum/rulesets/custom-rulesets/#anatomy-of-a-rule)

---

`v0.20`: **Support for auto fixing custom rules**

Got some rules that don't really need a human to look at?

Well now you can define an `AutoFixFunction` for your rules, and when you run with the `--fix` flag, the fixes will be applied to the file, or use `--fix-file` to write them to a different file.

See [Auto-Fixing Rule Violations](#auto-fixing-rule-violations) for more specifics.

---

`v0.19`: **Ignore rules with `x-lint-ignore`**

Got an error in your spec you know about but can't get round to fixing yet?
Migrating from zally and wanting to keep your existing `x-zally-ignore` issues silenced?

Now you can! Just add `x-lint-ignore: rule-id` to the yaml node reporting the failure (or `x-lint-ignore: [rule-one, rule-two]` if there are multiple issues to ignore).

---

`v0.18`: **New dashboard, new lint command, new rules!**.

Upgrades all around. There is a completely new `dashboard` command with a completely new dashboard terminal UI. It's 
completely interactive and allows you to explore, and filter violations, view full docs and see code. The `dashboard` command
also adds a new `-w` / `--watch` flag that will watch your OpenAPI file for changes and re-lint and re-render results automatically.

A re-written `lint` command that has a whole new rendering engine and output. Everything is much more readable, 
easier to see on a screen, matches the new `dashboard` style. It's 100% backwards compatible with previous versions, all flags as they were. 

New rules:

 - [no-request-body](https://quobix.com/vacuum/rules/operations/no-request-body/) - Ensures `GET` and `DELETE` operations do not have request bodies.
 - [duplicate-paths](https://quobix.com/vacuum/rules/operations/duplicate-paths/) - Ensures there are no duplicate paths exist
 - [no-unnecessary-combinator](https://quobix.com/vacuum/rules/schemas/no-unnecessary-combinator/) - Ensures no `allOf`, `oneOf` or `anyOf` combinators exist with a single schema inside them.
 - [camel-case-properties](https://quobix.com/vacuum/rules/schemas/camel-case-properties/) - Ensures all schema properties are `camelCase`.

---

`v0.17`: **Github Action**.

vacuum now has an official Github Action. [Read the docs](https://quobix.com/vacuum/github-action/), or check it out
in the [GitHub Marketplace](https://github.com/marketplace/actions/vacuum-openapi-linter-and-quality-analysis-tool).

---


`v0.16.11`: **Composed bundling mode**.

A different way to bundle exploded OpenAPI specifications into a single file. [Read the docs](https://quobix.com/vacuum/commands/bundle/).

---

`v0.16+` : **RFC 9535 Compliant**.

`v0.21+` : **JSON Path Plus Support**.

vacuum now supports both [RFC 9535](https://www.rfc-editor.org/rfc/rfc9535) JSONPath and **JSON Path Plus** extensions - full Spectral compatibility!

---

`v0.15+`: **Fixes, New Rules, Functions and Command**.

There is a new command `generate-ignorefile` that will generate an ignore file from a linting report.

New rule `no-request-body` checks for incorrect request bodies in operations, and `path-item-refs` checks for
$refs being used in path items.

---

### [Quick Start Guide 🚀](https://quobix.com/vacuum/start)

See all the documentation at https://quobix.com/vacuum

- [Installing vacuum](https://quobix.com/vacuum/installing/)
- [About vacuum](https://quobix.com/vacuum/about/)
- [Why should you care?](https://quobix.com/vacuum/why/)
- [Concepts](https://quobix.com/vacuum/concepts/)
- [FAQ](https://quobix.com/vacuum/faq/)
- [CLI Commands](https://quobix.com/vacuum/commands/)
  - [lint](https://quobix.com/vacuum/commands/lint/)
  - [vacuum report](https://quobix.com/vacuum/commands/report/)
  - [dashboard](https://quobix.com/vacuum/commands/dashboard/)
  - [html-report](https://quobix.com/vacuum/commands/html-report/)
  - [bundle](https://quobix.com/vacuum/commands/bundle/)
  - [spectral-report](https://quobix.com/vacuum/commands/spectral-report/)
  - [language-server](https://quobix.com/vacuum/commands/language-server/)
  - [Change Detection](https://quobix.com/vacuum/commands/change-detection/)
- [Developer API](https://quobix.com/vacuum/api/getting-started/)
  - [Using The Index](https://quobix.com/vacuum/api/spec-index/)
  - [RuleResultSet](https://quobix.com/vacuum/api/rule-resultset/)
  - [Loading a RuleSet](https://quobix.com/vacuum/api/loading-ruleset/)
  - [Linting Non-OpenAPI Files](https://quobix.com/vacuum/api/non-openapi/)
  - [Custom Golang Functions](https://quobix.com/vacuum/api/custom-functions/)
  - [Custom JavaScript Functions](https://quobix.com/vacuum/api/custom-javascript-functions/)
- [Rules](https://quobix.com/vacuum/rules/)
  - [Examples](https://quobix.com/vacuum/rules/examples/)
  - [Tags](https://quobix.com/vacuum/rules/tags/)
  - [Descriptions](https://quobix.com/vacuum/rules/descriptions/)
  - [Schemas](https://quobix.com/vacuum/rules/schemas/)
  - [Spec Information](https://quobix.com/vacuum/rules/information/)
  - [Operations & Paths](https://quobix.com/vacuum/rules/operations/)
  - [Validation](https://quobix.com/vacuum/rules/validation/)
  - [Security](https://quobix.com/vacuum/rules/security/)
  - [OWASP](https://quobix.com/vacuum/rules/owasp/)
- [Functions](https://quobix.com/vacuum/functions/)
  - [Core Functions](https://quobix.com/vacuum/functions/core/) 
  - [OpenAPI Functions](https://quobix.com/vacuum/functions/openapi/)
  - [OWASP Functions](https://quobix.com/vacuum/functions/owasp/)
- [Understanding RuleSets](https://quobix.com/vacuum/rulesets/understanding/)
  - [Sharing RuleSets](https://quobix.com/vacuum/rulesets/sharing/)
  - [All Rules](https://quobix.com/vacuum/rulesets/all/)
  - [No Rules](https://quobix.com/vacuum/rulesets/no-rules/)
  - [Recommended Rules](https://quobix.com/vacuum/rulesets/recommended/)
  - [Custom Rules](https://quobix.com/vacuum/rulesets/custom-rulesets/)

---

> **vacuum can suck all the lint of a 5mb OpenAPI spec in milliseconds.**

Designed to reliably lint OpenAPI specifications, **very, very quickly**. Including _very large_ ones. Spectral can be quite slow
when used as an API and does not scale for enterprise applications.

vacuum will tell you what is wrong with your spec, why, where, and how to fix it. 

vacuum will work at scale and is designed as a CLI (with a web or console UI) and a library to be consumed in other applications.

### Dashboard

vacuum comes with an interactive dashboard (`vacuum dashboard <your-openapi-spec.yaml>`) allowing you to explore
rules and violations in a console, without having to scroll through thousands of results.

<a href="https://quobix.com/vacuum/commands/dashboard/">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/assets/dashboard.gif">
  <img alt="speakeasy'" src=".github/sponsors/speakeasy-github-sponsor-light.svg">
</picture>
</a>

To read about the dashboard, see the [dashboard command docs](https://quobix.com/vacuum/commands/dashboard/).

### HTML Report

vacuum can generate an easy to navigate and understand HTML report. Like the dashboard
you can explore broken rules and violations, but in your browser.

No external dependencies, the HTML report will run completely offline.

![vacuum html-report](html-report-screenshot.png)

---

> **_Supports OpenAPI Version 2 (Swagger) and Version 3+_**

You can use either **YAML** or **JSON**, vacuum supports both formats.

## Using vacuum with pre-commit

Vacuum can be used with [pre-commit](https://pre-commit.com).

To do that, add to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/daveshanley/vacuum
    rev: # a tag or a commit hash from this repo, see https://github.com/daveshanley/vacuum/releases
    hooks:
      - id: vacuum
```

See the [hook definition](./.pre-commit-hooks.yaml) here for details on what options the hook uses and what files it checks by default.

If no filenames or more than one filename in your repository matches the default `files` pattern in the hook definition,
the pattern needs to be overridden in your config so that it matches exactly one filename to lint at a time.
To lint multiple files, specify the hook multiple times with the appropriate overrides.

## Build an interactive HTML report 

```
./vacuum html-report <your-openapi-spec.yaml | vacuum-report.json.gz> <report-name.html>
```

You can replace `report-name.html` with your own choice of filename. Open the report
in your favorite browser and explore the results. 


## See full linting report 

```
./vacuum lint -d <your-openapi-spec.yaml>
```


## Lint multiple files at once

```
./vacuum lint -d <spec1.yaml> <spec2.yaml> <spec3.yaml>
```

## Lint multiple files using a glob pattern

```
./vacuum lint -d some/path/**/*.yaml
```

## See full linting report with inline code snippets

```
./vacuum lint -d -s <your-openapi-spec.yaml>
```

## See just the linting errors

```
./vacuum lint -d -e <your-openapi-spec.yaml>
```

## See just a specific category of report


```
./vacuum lint -d -c schemas <your-openapi-spec.yaml>
```

The options here are:

- `examples`
- `operations`
- `information`
- `descriptions`
- `schemas`
- `security`
- `tags`
- `validation`
- `owasp`

## Generate a Spectral compatible report

If you're already using Spectral JSON reports, and you want to use vacuum instead, use the `spectral-report` command

```
./vacuum spectral-report <your-openapi-spec.yaml> <report-output-name.json>
```

The report file name is _optional_. The default report output name is `vacuum-spectral-report.json`


## Generate a `vacuum report`

Vacuum reports are complete snapshots in time of a linting report for a specification. These reports can be 'replayed' 
back through vacuum. Use the `dashboard` or the `html-report` commands to 'replay' the report and explore the results
as they were when the report was generated.

```
./vacuum report -c <your-openapi-spec.yaml> <report-prefix>
```

The default name of the report will be `vacuum-report-MM-DD-YY-HH_MM_SS.json`. You can change the prefix by supplying
it as the second argument to the `report` command. 

Ideally, **you should compress the report using `-c`**. This shrinks down the size significantly. vacuum automatically
recognizes a compressed report file and will deal with it automatically when reading.

> When using compression, the file name will be `vacuum-report-MM-DD-YY-HH_MM_SS.json.gz`. vacuum uses gzip internally.

## Ignoring specific linting errors

You can ignore specific linting errors by providing an `--ignore-file` argument to the `lint` and `report` commands.

```
./vacuum lint --ignore-file <path-to-ignore-file.yaml> -d <your-openapi-spec.yaml>
```

```
./vacuum report --ignore-file <path-to-ignore-file.yaml> -c <your-openapi-spec.yaml> <report-prefix>
```

The ignore-file should point to a .yaml file that contains a list of errors to be ignored by vacuum. The structure of the
yaml file is as follows:

```
<rule-id-1>:
  - <json_path_to_error_or_warning_1>
  - <json_path_to_error_or_warning_2>
<rule-id-2>:
  - <json_path_to_error_or_warning_1>
  - <json_path_to_error_or_warning_2>
  ...
```

Ignoring errors is useful for when you want to implement new rules to existing production APIs. In some cases, 
correcting the lint errors would result in a breaking change. Having a way to ignore these errors allows you to implement
the new rules for new APIs while maintaining backwards compatibility for existing ones.

---

## Try out the dashboard

This is an early, but working console UI for vacuum. The code isn't great, it needs a lot of clean up, but
if you're interested in seeing how things are progressing, it's available.

```
./vacuum dashboard <your-openapi-spec.yaml | vacuum-report.json.gz>
```

---
## Supply your own Spectral compatible ruleset

If you're already using Spectral and you have your own [custom ruleset](https://meta.stoplight.io/docs/spectral/e5b9616d6d50c-custom-rulesets#custom-rulesets),
then you can use it with vacuum! 

The `lint`, `dashboard` and `spectral-report` commands all accept a `-r` or `--ruleset` flag, defining the path to your ruleset file.

### Here are some examples you can try

**_All rules turned off_**
```
./vacuum lint -r rulesets/examples/norules-ruleset.yaml <your-openapi-spec.yaml>
```

**_Only recommended rules_**
```
./vacuum lint -r rulesets/examples/recommended-ruleset.yaml <your-openapi-spec.yaml>
```

**_Enable specific rules only_**
```
./vacuum lint -r rulesets/examples/specific-ruleset.yaml <your-openapi-spec.yaml>
```

**_Custom rules_**
```
./vacuum lint -r rulesets/examples/custom-ruleset.yaml <your-openapi-spec.yaml>
```

**_All rules, all of them!**
```
./vacuum lint -r rulesets/examples/all-ruleset.yaml <your-openapi-spec.yaml>
```

---

## Configuration

### File
You can configure vacuum using a configuration file named `vacuum.conf.yaml`

By default, vacuum searches for this file in the following directories
1. Working directory
2. `$XDG_CONFIG_HOME`
3. `${HOME}/.config`

You can also specify a path to a file using the `--config` flag

Global flags are configured as top level nodes
```yaml
time: true
base: 'http://example.com'
...
```
Command specific flags are configured under a node with the commands name
```yaml
...
lint:
  silent: true
  ...
```

### Environmental variables

You can configure global vacuum flags using environmental variables in the form of: `VACUUM_<flag>`

If a flag, has a `-` in it, replace with `_`


## Auto-fixing rule violations

If you have a rule that doesn't need a human to look at it, and the change can be reliably automated you can configure an `AutoFixFunction` on the rule. When you then run the `lint` command you can pass the `--fix` flag and the violation will be automatically fixed.

### Set up

1. Define a rule that has an `autoFixFunction`, e.g.:
```yaml
rules:
  use-compatible-extensions:
    autoFixFunction: useExtensibleEnum
    description: Prefer compatible extensions
    id: use-compatible-extensions
    given: "$.components.schemas[?@.enum]"
    severity: warn
    message: Use x-extensible-enum instead of enum for better compatibility
    then:
      field: enum
      function: falsy
```

This rule flags any usage of `enum` and recommends they are updated to `x-extensible-enum`.
A simple change which can be easily auto fixed!

2. Create a function which performs the auto-fix.
```go
func useExtensibleEnum(
	node *yaml.Node,
	document *yaml.Node,
	context *model.RuleFunctionContext,
) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return node, nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}

		keyNode := node.Content[i]

		if keyNode.Value == "enum" {
			keyNode.Value = "x-extensible-enum"

			return node, nil
		}
	}

	return node, nil
}
```

> [!NOTE]
> The auto fix function must satisfy the `AutoFixFunction` type.
> It should take in the `*yaml.Node` of the violation, the root `*yaml.Node` of the document and the `RuleFunctionContext`.
> It should return the fixed `*yaml.Node` and an error.

3. Configure your `RuleSetExecution` to use the auto fix function.
```go
func Lint(rulesFile string, specFile string) error {
	rules, err := rulesets.LoadLocalRuleSet(ctx, rulesFile)
	if err != nil {
		return fmt.Errorf("error loading ruleset: %w", err)
	}

	rs := rulesets.BuildDefaultRuleSetsWithLogger(slog.Logger).
		GenerateRuleSetFromSuppliedRuleSet(rules)

	// NOTE: only showing the fields on the RuleSetExecution relevant to auto-fixing.
	results := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		AutoFixFunctions: map[string]model.AutoFixFunction{
			"useExtensibleEnum": useExtensibleEnum,
		},
		ApplyAutoFixes:         true,
		RuleSet:                rs,
	})

	// Write back to file if fixes were applied
	if len(lintResults.FixedResults) > 0 && autoFix {
		fileInfo, _ := os.Stat(specFile)

		err = os.WriteFile(specFile, result.ModifiedSpec, fileInfo.Mode())
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", c.file, err)
		}
	}

	return nil
}
```

When the auto fix function runs, if it returns an error the fix will not be applied, the error will be logged, and the violation will be reported in the standard results.

If the auto fix function succeeds the yaml node flagged by the violation will be replaced with the transformed version returned by the auto fix function.

> [!TIP]
> When using `vacuum` as a library You can access the fixed yaml content in the `RuleSetExecutionResult.ModifiedSpec`, and choose where to write the file.
> 
> When using `vacuum` as a cli, the `--fix` flag will overwrite the spec file in place, and `--fix-file` flag lets you specify an alternative file to write the content to, if you want to compare the outputs.

### Usage


> Logo gopher is modified, originally from [egonelbre](https://github.com/egonelbre/gophers)
