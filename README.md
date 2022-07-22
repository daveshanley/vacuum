![vacuum logo](logo.png)

# vacuum - The world's fastest OpenAPI & Swagger linter.

![Pipeline](https://github.com/daveshanley/vacuum/workflows/vaccum%20pipeline/badge.svg)
[![codecov](https://codecov.io/gh/daveshanley/vacuum/branch/main/graph/badge.svg?)](https://codecov.io/gh/daveshanley/vacuum)
[![Go Report Card](https://goreportcard.com/badge/github.com/daveshanley/vacuum)](https://goreportcard.com/report/github.com/daveshanley/vacuum)

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


## Documentation

### [Quick Start Guide 🚀](https://quobix.com/vacuum/start)

See all the documentation at https://quobix.com/vacuum

- [Installing vacuum](https://quobix.com/vacuum/installing/)
- [About vacuum](https://quobix.com/vacuum/about/)
- [Why should you care?](https://quobix.com/vacuum/why/)
- [Concepts](https://quobix.com/vacuum/concepts/)
- [CLI Commands](https://quobix.com/vacuum/cli/)
  - [lint](https://quobix.com/vacuum/cli/lint) 
  - [vacuum report](https://quobix.com/vacuum/cli/report)
  - [dashboard](https://quobix.com/vacuum/cli/dashboard)
  - [html-report](https://quobix.com/vacuum/cli/html-report)
  - [spectral-report](https://quobix.com/vacuum/cli/spectral-report)
- [Developer API](https://quobix.com/vacuum/api/getting-started/)
  - [Using The Index](https://quobix.com/vacuum/api/spec-index/)
  - [RuleResultSet](https://quobix.com/vacuum/api/rule-resultset/)
  - [Loading a RuleSet](https://quobix.com/vacuum/api/loading-ruleset/)
- [Rules](https://quobix.com/vacuum/rules/)
  - [Examples](https://quobix.com/vacuum/rules/examples/)
  - [Tags](https://quobix.com/vacuum/rules/tags/)
  - [Descriptions](https://quobix.com/vacuum/rules/descriptions/)
  - [Schemas](https://quobix.com/vacuum/rules/schemas/)
  - [Spec Information](https://quobix.com/vacuum/rules/information/)
  - [Operations & Paths](https://quobix.com/vacuum/rules/operations/)
  - [Validation](https://quobix.com/vacuum/rules/validation/)
  - [Security](https://quobix.com/vacuum/rules/security/)
- [Functions](https://quobix.com/vacuum/functions/)
  - [Core Functions](https://quobix.com/vacuum/functions/core/) 
  - [OpenAPI Functions](https://quobix.com/vacuum/functions/openapi/)
- [Understanding RuleSets](https://quobix.com/vacuum/rulesets/understanding/)
  - [All Rules](https://quobix.com/vacuum/rulesets/all/)
  - [No Rules](https://quobix.com/vacuum/rulesets/no-rules/)
  - [Recommended Rules](https://quobix.com/vacuum/rulesets/recommended/)
  - [Custom Rules](https://quobix.com/vacuum/rulesets/custom-rulesets/)


---

> **vacuum can suck all the lint of a 5mb OpenAPI spec in about 250ms.**

Designed to reliably lint OpenAPI specifications, **very, very quickly**. Including _very large_ ones. Spectral can be quite slow
when used as an API and does not scale for enterprise applications.

vacuum will tell you what is wrong with your spec, why, where and how to fix it. 

vacuum will work at scale and is designed as a CLI (with a UI) and a library to be consumed in other applications.

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

You can use either **YAML** or **JSON** vacuum supports both.

AsyncAPI support is coming soon!

## Check out the code

```
git clone https://github.com/daveshanley/vacuum.git
```
### Change directory into `vacuum`

```
cd vacuum
```

## Build the code

```
go build vacuum.go
```

## Run the code

```
./vacuum lint <your-openapi-spec.yaml>
```

## Running vacuum via Docker

vacuum is available as a container, you can pull the image from
[Docker Hub](https://hub.docker.com/repository/docker/dshanley/vacuum/general)

```
docker pull dshanley/vacuum
```

or you can pull it from [Github packages](https://github.com/daveshanley/vacuum/pkgs/container/vacuum).

```
docker pull ghcr.io/daveshanley/vacuum:latest
```

To run, just add `docker dshanley/vacuum` as your command, like so

```
docker dshanley/vacuum lint <your-openapi-spec.yaml>
```

---
> 👉 **Please note, the flags and commands below will change as the experience is refined.** 👈
---

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
## Supply your own Spectral ruleset

If you're already using Spectral and you have your own [custom ruleset](https://meta.stoplight.io/docs/spectral/e5b9616d6d50c-custom-rulesets#custom-rulesets),
then you can use it with Vacuum! 

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

Let me know what you think.

> Logo gopher is modified, originally from [egonelbre](https://github.com/egonelbre/gophers)