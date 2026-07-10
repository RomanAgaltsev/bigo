# bigo (big O)

[![lint](https://github.com/RomanAgaltsev/bigo/actions/workflows/lint.yml/badge.svg)](https://github.com/RomanAgaltsev/bigo/actions/workflows/lint.yml)
[![test](https://github.com/RomanAgaltsev/bigo/actions/workflows/test.yml/badge.svg)](https://github.com/RomanAgaltsev/bigo/actions/workflows/test.yml)
[![security](https://github.com/RomanAgaltsev/bigo/actions/workflows/security.yml/badge.svg)](https://github.com/RomanAgaltsev/bigo/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/RomanAgaltsev/bigo/branch/main/graph/badge.svg)](https://codecov.io/gh/RomanAgaltsev/bigo)
[![Go Reference](https://pkg.go.dev/badge/github.com/RomanAgaltsev/bigo.svg)](https://pkg.go.dev/github.com/RomanAgaltsev/bigo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go static analyzer that infers the **asymptotic time complexity** of Go code
and checks it against developer-declared budgets:

```go
//bigo:max O(n log n)
func Search(xs []int, target int) int { ... }
```

bigo fills the empty *asymptotic* slot in the golangci-lint complexity family:
gocyclo/gocognit/cyclop measure structural complexity; bigo measures **growth**.

## The three-valued model

Every budgeted function gets exactly one of three verdicts:

- **within** — the inferred bound is provably inside the budget. Silent.
- **exceeds** — provably over budget. Diagnostic.
- **unverifiable** — bigo hit something it cannot see through (an interface
  call, a closure, recursion, a `go` statement). Diagnostic naming the exact
  blocker. bigo **never guesses**: code it cannot analyze is reported as
  unverifiable, not silently assumed cheap.

Resolve `unverifiable` by asserting what bigo cannot infer:

```go
//bigo:cost O(log n)      // assert a callee's (or interface method's) cost
func lookup(keys []string, k string) int

//bigo:ignore             // trust a helper entirely: treat as O(1)
func metrics(x int) int
```

## Directives

| Directive | Meaning |
|---|---|
| `//bigo:max O(...)` | budget: diagnostic on exceed / unverifiable |
| `//bigo:max O(n*m) where n=len(a), m=len(b)` | multi-size budget with bindings |
| `//bigo:cost O(...)` | assert the cost of a function or interface method |
| `//bigo:ignore` | trust: treat as O(1) |
| `//bigo:space O(...)` | reserved (Phase 2); parsed but inert |

Directives use the `//go:` shape — no space after `//`. A malformed directive
is a diagnostic, never silently ignored.

## Install & run

```sh
go install github.com/RomanAgaltsev/bigo/cmd/bigo@latest

bigo ./...            # check budgets (CI-friendly exit code)
bigo -report ./...    # print the inferred bound of every analyzable function
```

## Use with golangci-lint

bigo ships as a golangci-lint **module plugin**; build a custom binary once:

```yaml
# .custom-gcl.yml
version: v2.12.2
name: custom-gcl
destination: ./bin
plugins:
  - module: github.com/RomanAgaltsev/bigo
    import: github.com/RomanAgaltsev/bigo/plugin
```

```yaml
# .golangci.yml
linters:
  enable:
    - bigo
  settings:
    custom:
      bigo:
        type: module
        description: Checks //bigo:max asymptotic complexity budgets.
```

```sh
golangci-lint custom && ./bin/custom-gcl run
```

## What bigo does not count (yet)

Each can only *miss* a violation, never invent one:

- `append` is amortized O(1) (including `append(a, b...)`), `make` is O(1).
- String concatenation and comparison are O(1) per operation.
- Map index/assign/delete are O(1).
- Recursion, interface calls without `//bigo:cost`, closures, and
  `range`-over-func iterators are **unverifiable** (in progress).
- Cross-package calls resolve only through the curated stdlib cost table or
  your `//bigo:cost` annotations.

## Status & versioning

Complete: intraprocedural engine, cost tables, acyclic interprocedural
summaries, generics at instantiation, golangci-lint plugin.
The **analysis surface is pre-stable**: verdicts may
change between minor versions as inference improves. Design-complete, not built recursion,
interface resolution, space complexity.

## License

MIT
