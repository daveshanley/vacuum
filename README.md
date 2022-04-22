# vacuum - The world's fastest OpenAPI & Swagger linter.
![Pipeline](https://github.com/daveshanley/vacuum/workflows/vaccum%20pipeline/badge.svg)
[![codecov](https://codecov.io/gh/daveshanley/vacuum/branch/main/graph/badge.svg?)](https://codecov.io/gh/daveshanley/vacuum)
[![Go Report Card](https://goreportcard.com/badge/github.com/daveshanley/vacuum)](https://goreportcard.com/report/github.com/daveshanley/vacuum)

An **ultra-super-fast**, lightweight OpenAPI linter and quality checking tool, inspired by [Spectral](https://github.com/stoplightio/spectral).

It's also compatible with existing [Spectral](https://github.com/stoplightio/spectral) rulesets.

---

> **vacuum can suck all the lint of a 5mb OpenAPI spec in about 250ms.**

Designed to reliably lint OpenAPI specifications, **very, very quickly**. Including _very large_ ones. Spectral can be quite slow
when used as an API and does not scale for enterprise applications.

Vacuum will tell you what is wrong with your spec, why, where and how to fix it. 

Vacuum will work at scale and is designed as a CLI and a library to be consumed in other applications.

If you want to try it out in its **earliest stages**

> Please be warned, this is _early_ code. I am actively working on it.
>> **_Supports OpenAPI Version 2 (Swagger) and Version 3+_**

You can use either **YAML** or **JSON** vacuum supports both.

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
---
> 👉 **Please note, the flags and commands below will change as the experience is refined.** 👈
---

## See full linting report details

```
./vacuum lint -d <your-openapi-spec.yaml>
```

## See full linting report with inline code snippets

```
./vacuum lint -d -s <your-openapi-spec.yaml>
```

## See just the linting errors

```
./vacuum -d -e <your-openapi-spec.yaml>
```

## See just a specific category of report


```
./vacuum -d -c schemas <your-openapi-spec.yaml>
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

If you're already using Spectral JSON reports, and you want to use vacuum instead, use the `report` command

```
./vacuum report <your-openapi-spec.yaml> <report-output-name.json>
```

The report file name is _optional_. The default report output name is `vacuum-spectral-report.json`


## Try out the dashboard

This is a total mess at the moment, but you can see a glimpse of the future.
Don't rely on this yet, it's not accurate.

```
./vacuum dashboard <your-openapi-spec.yaml>
```



Let me know what you think.