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
| `//bigo:space O(...)` | space budget: heap (total allocated) + recursion stack |

A declaration may carry more than one directive — `//bigo:cost` tells callers
what this function costs while `//bigo:max` gates its own body:

```go
//bigo:cost O(1)
//bigo:max O(n)
func Lookup(keys []string, k string) int { ... }
```

Directives use the `//go:` shape — no space after `//`. A malformed directive
is a diagnostic, never silently ignored.

Size variables can name receiver/parameter fields when bigo can prove the
field is not mutated between function entry and the loop:

```go
//bigo:max O(n) where n=len(s.items)
func (s *S) Sum() int { ... }
```

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

## Recursion

bigo solves self-recursive functions whose argument provably shrinks toward a
base case, in three recurrence families:

- **Subtractive** — `T(n) = T(n−c) + f(n)` → `O(n·f(n))`. Recursive scans and
  guarded countdowns (`sum(xs[1:])`, `f(n-1)`).
- **Master** — `T(n) = a·T(n/b) + f(n)` for a uniform divisor `b`. Binary
  search (`O(log n)`), balanced divide-and-conquer (`O(n)`, `O(n log n)`).
- **Akra–Bazzi** — `T(n) = Σ aᵢ·T(n/bᵢ) + f(n)` for unbalanced integer-ratio
  splits, when the critical exponent `p` (solving `Σ aᵢ·bᵢ^−p = 1`) is an
  integer.

A bound is emitted only when the measure — a slice length or an integer
magnitude — **provably strictly decreases** toward a base; a wrong answer here
would be a wrong bound on possibly non-terminating code. These stay
unverifiable (⊤):

- **Pointer-structure recursion** (walking a `*Node` tree or list): no
  slice/integer measure to decrease.
- **Non-terminating or growing recursion**: `f(n-1)` with no base guard, `f(n+1)`.
- **Divisive recursion whose measure can reach 0 without a base**: `f(n/2)`
  guarded only by `n >= 0` (integer division truncates toward zero, so `0/2 == 0`
  is a fixed point), or `f(xs[:len/2])` with no empty-slice base (`xs[:0]` stays
  empty). A divisive step graduates only when the recursing side proves the
  measure `>= 1` — an `n > 0` guard, or an `n == 0` / `len(xs) == 0` base.
- **Exponential recurrences**: naive Fibonacci (`T(n-1)+T(n-2)`).
- **Non-integer critical exponents**: `2·T(n/4)` (exponent ½).
- **Non-constant multiplicity** (self-calls under a size loop), mutual and
  multi-function recursion, and per-level work whose cost depends on the
  recursion's *results* (merge sort's `merge(l, r)`, which would need relational
  length tracking).

## Space budgets

`//bigo:space O(...)` gates a function's asymptotic **space**, split into two
soundness classes that bigo treats asymmetrically:

- **Stack** — the peak recursion depth, from the same recurrence solver used for
  time (`O(n)` subtractive, `O(log n)` divisive). This is a *true* peak, so it
  proves both `within` and `exceeds`.
- **Heap** — an **upper bound on peak** live memory, modeled as *total
  allocated* (`make([]T, n)` → `O(n)`, `append(a, b...)` → `O(len(b))`, one
  allocation × its enclosing-loop trips). Because total allocated over-counts a
  peak that the garbage collector shrinks, heap proves `within` **only** — never
  `exceeds`.

The consequence is deliberate: **a space budget never reports a false
`exceeds`.** A function that allocates `O(1)` inside an `n`-loop has peak heap
`O(1)` but total-allocated `O(n)`; against an `O(1)` budget bigo reports
`cannot verify` (annotate to resolve), not `exceeds`, because bigo bounds total
allocation as a safe over-approximation of peak:

```go
//bigo:space O(n)
func RecSum(xs []int) int { // stack O(len(xs)); within
	if len(xs) == 0 {
		return 0
	}
	return xs[0] + RecSum(xs[1:])
}
```

Concurrent allocation (`go`) and calls whose space is unknown are `⊤`
(unverifiable), exactly as on the time axis.

## What bigo does not count (yet)

Each can only *miss* a violation, never invent one:

- `append` is amortized O(1) (including `append(a, b...)`), `make` is O(1).
- String concatenation and comparison are O(1) per operation.
- Map index/assign/delete are O(1).
- Self-recursion over a size measure is solved (see [Recursion](#recursion));
  interface calls without `//bigo:cost`, closures, and `range`-over-func
  iterators remain **unverifiable** (in progress).
- Cross-package calls resolve only through the curated stdlib cost table or
  your `//bigo:cost` annotations.
- Field-size stability assumes no data race on the analyzed object (the Go
  memory model makes racy reads undefined anyway). Channel-typed fields are
  never used as sizes: channel synchronization makes concurrent mutation
  legal, so `len(ch)` has no stable entry value.
- Trip counts cover counted loops (increasing/decreasing with constant
  steps), geometric growth/decay, `range` over slices/maps/strings, and
  two-variable bisection. Worklist loops (`for len(queue) > 0`) and pointer
  chasing remain unverifiable; size-measure self-recursion is solved.
- Loop-bound arithmetic assumes values bounded by real memory; the
  `(lo+hi)/2` bisection form additionally assumes `lo+hi` does not overflow
  (requires a length above 2^62). The `lo + (hi-lo)/2` form needs no
  assumption.

## Status & versioning

Complete: intraprocedural engine, cost tables, acyclic interprocedural
summaries, generics at instantiation, golangci-lint plugin, size-measure
recurrence solving (subtractive / Master / Akra–Bazzi).
The **analysis surface is pre-stable**: verdicts may
change between minor versions as inference improves. Design-complete but not yet
built: interface resolution, space complexity.

## License

MIT
