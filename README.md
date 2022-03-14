# vacuum - OpenAPI linter for golang
[![codecov](https://codecov.io/gh/daveshanley/vacuum/branch/main/graph/badge.svg?)](https://codecov.io/gh/daveshanley/vacuum)
[![Go Report Card](https://goreportcard.com/badge/github.com/daveshanley/vacuum)](https://goreportcard.com/report/github.com/daveshanley/vacuum)

A super-fast, lightweight OpenAPI linter, inspired by [Spectral](https://github.com/stoplightio/spectral).

---



Designed to reliably and lint OpenAPI specifications, **very quickly**. Including _very large_ ones. Spectral can be quite slow
when used as an API and does not scale at all for enterprise applications.

Vacuum will tell you what is wrong with your spec, why, where and how to fix it. 

Vacuum will work at scale and is designed as a CLI and a library to be consumed in other applications.

If you want to try it out in its **earliest stage**

> Please be warned, this is _alpha_ code. It's
> very early code and will be a bit wrong sometimes.
>> **_Only works with OpenAPI 3 correctly right now_**


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

Let me know what you think.