![logo](logo.png)

# vacuum - The world's fastest OpenAPI & Swagger linter.

![Pipeline](https://github.com/daveshanley/vacuum/workflows/vaccum%20pipeline/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/daveshanley/vacuum)](https://goreportcard.com/report/github.com/daveshanley/vacuum)
[![discord](https://img.shields.io/discord/923258363540815912)](https://discord.gg/UAcUF78MQN)
[![Docs](https://img.shields.io/badge/godoc-reference-5fafd7)](https:/-/pkg.go.dev/github.com/daveshanley/vacuum)
[![GitHub downloads](https://img.shields.io/github/downloads/daveshanley/vacuum/total?label=github%20downloads&style=flat-square)](https://github.com/daveshanley/vacuum/releases)
[![npm](https://img.shields.io/npm/dm/@quobix/vacuum?style=flat-square&label=npm%20downloads)](https://www.npmjs.com/package/@quobix/vacuum)
[![Docker Pulls](https://img.shields.io/docker/pulls/dshanley/vacuum?style=flat-square)](https://hub.docker.com/r/dshanley/vacuum)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)

An **ultra-super-fast**, lightweight OpenAPI linter and quality checking tool, written in golang and inspired by [Spectral](https://github.com/stoplightio/spectral).

It's also compatible with existing [Spectral](https://github.com/stoplightio/spectral) rulesets.

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

```
curl -fsSL https://quobix.com/scripts/install_vacuum.sh | sh
```

## Install using [Docker](https://hub.docker.com/r/dshanley/vacuum)

The image is available at: https://hub.docker.com/r/dshanley/vacuum

```
docker pull dshanley/vacuum
```

To run, mount the current working dir to the container and use a relative path to your spec, like so

```
docker run --rm -v $PWD:/work:ro dshanley/vacuum lint <your-openapi-spec.yaml>
```
Alternatively, you can pull it from
[Github packages](https://github.com/daveshanley/vacuum/pkgs/container/vacuum).
To do that, replace `dshanley/vacuum` with `ghcr.io/daveshanley/vacuum` in the above commands.


---


## Sponsors
If your company is using `vacuum`, please considering [supporting this project](https://github.com/sponsors/daveshanley),
like our _very kind_ sponsors:


<a href="https://speakeasyapi.dev/?utm_source=vacuum+repo&utm_medium=github+sponsorship">
<picture style="width: 80%">
  <source media="(prefers-color-scheme: dark)" srcset=".github/sponsors/speakeasy-github-sponsor-dark.svg">
  <img alt="speakeasy'" src=".github/sponsors/speakeasy-github-sponsor-light.svg">
</picture>
</a>

[Speakeasy](https://speakeasyapi.dev/?utm_source=vacuum+repo&utm_medium=github+sponsorship)

---

## Come chat with us

Need help? Have a question? Want to share your work? [Join our discord](https://discord.gg/UAcUF78MQN) and
come say hi!

## Documentation

---
🔥 **New in** `v0.3.0+` 🔥 : [Custom JavaScript Functions](https://quobix.com/vacuum/api/custom-javascript-functions/) are now available out of the box.

Write custom functions in JavaScript and use them in any ruleset. No need
to compile golang code to extend vacuum anymore!

[Learn more about building custom JavaScript functions](https://quobix.com/vacuum/api/custom-javascript-functions/).


---
**New in** `v0.2.0+`: [OWASP API rules](https://quobix.com/vacuum/rules/owasp/) are now available out of the box.

[Learn more about enabling OWASP API rules](https://quobix.com/vacuum/rulesets/owasp/).

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
  - [spectral-report](https://quobix.com/vacuum/commands/spectral-report/)
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
  - [All Rules](https://quobix.com/vacuum/rulesets/all/)
  - [No Rules](https://quobix.com/vacuum/rulesets/no-rules/)
  - [Recommended Rules](https://quobix.com/vacuum/rulesets/recommended/)
  - [Custom Rules](https://quobix.com/vacuum/rulesets/custom-rulesets/)

---

> **vacuum can suck all the lint of a 5mb OpenAPI spec in about 230ms.**

Designed to reliably lint OpenAPI specifications, **very, very quickly**. Including _very large_ ones. Spectral can be quite slow
when used as an API and does not scale for enterprise applications.

vacuum will tell you what is wrong with your spec, why, where and how to fix it. 

vacuum will work at scale and is designed as a CLI (with a web or console UI) and a library to be consumed in other applications.

### Dashboard

vacuum comes with an interactive dashboard (`vacuum dashboard <your-openapi-spec.yaml>`) allowing you to explore
rules and violations in a console, without having to scroll through thousands of results.

![vacuum dashboard](dashboard-screenshot.png)

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

> Logo gopher is modified, originally from [egonelbre](https://github.com/egonelbre/gophers)
