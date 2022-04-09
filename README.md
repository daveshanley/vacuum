# vacuum - The world's fastest OpenAPI linter.
[![codecov](https://codecov.io/gh/daveshanley/vacuum/branch/main/graph/badge.svg?)](https://codecov.io/gh/daveshanley/vacuum)
[![Go Report Card](https://goreportcard.com/badge/github.com/daveshanley/vacuum)](https://goreportcard.com/report/github.com/daveshanley/vacuum)

A **super-fast**, lightweight OpenAPI linter and quality checking tool, inspired by [Spectral](https://github.com/stoplightio/spectral).

It's also compatible with existing [Spectral](https://github.com/stoplightio/spectral) rulesets.

---

> **vacuum can suck all the lint of a 5mb OpenAPI spec in about 250ms.**

Designed to reliably and lint OpenAPI specifications, **very quickly**. Including _very large_ ones. Spectral can be quite slow
when used as an API and does not scale for enterprise applications.

Vacuum will tell you what is wrong with your spec, why, where and how to fix it. 

Vacuum will work at scale and is designed as a CLI and a library to be consumed in other applications.

If you want to try it out in its **earliest stages**

> Please be warned, this is _early_ code.
>> **_Supports OpenAPI Version 2 (Swagger) and Version 3+_**


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
./vacuum <your-openapi3-spec.yaml>
```
---
> 👉 **Please note, the flags and commands below will change as the experience is refined.** 👈
---

## See full linting report details

```
./vacuum <your-openapi3-spec.yaml> -d
```

## See full linting report with inline code snippets

```
./vacuum <your-openapi3-spec.yaml> -d -s
```

## See just the linting errors

```
./vacuum <your-openapi3-spec.yaml> -d -e
```

## See just a specific category of report


```
./vacuum <your-openapi3-spec.yaml> -d -c schemas
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




Let me know what you think.