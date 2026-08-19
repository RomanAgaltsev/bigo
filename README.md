# bigo (big O)

[![lint](https://github.com/RomanAgaltsev/bigo/actions/workflows/lint.yml/badge.svg)](https://github.com/RomanAgaltsev/bigo/actions/workflows/lint.yml)
[![test](https://github.com/RomanAgaltsev/bigo/actions/workflows/test.yml/badge.svg)](https://github.com/RomanAgaltsev/bigo/actions/workflows/test.yml)
[![security](https://github.com/RomanAgaltsev/bigo/actions/workflows/security.yml/badge.svg)](https://github.com/RomanAgaltsev/bigo/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/RomanAgaltsev/bigo/branch/main/graph/badge.svg)](https://codecov.io/gh/RomanAgaltsev/bigo)
[![Go Reference](https://pkg.go.dev/badge/github.com/RomanAgaltsev/bigo.svg)](https://pkg.go.dev/github.com/RomanAgaltsev/bigo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Quickstart:** [install](#install--run) · [golangci-lint](#use-with-golangci-lint)
· [GitHub Action](#in-ci) · [what bigo cannot bound](#what-bigo-does-not-count-yet)

bigo bounds **roughly a third** of hand-written functions in large real
codebases and reports the rest as ⊤ (unverifiable) — deliberately, because the
one rule it will not break is emitting a wrong bound.

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

## Trust files (`-trust`)

Most of a real repository is unverifiable because of code you cannot edit. A
trust file is where you assert bounds for that code, so your own functions get
verdicts and your budgets become enforceable.

### Finding out what to write

```
bigo trust init -o bigo.trust
```

This writes a starter file: the keys blocking the most of your functions,
ranked, every line commented out. **bigo suggests keys, never bounds** — the
placeholder does not parse, so an unedited line fails loudly rather than
asserting something nobody chose.

```
# 47 of your functions are blocked only by this.
# fmt.Sprintf O(...)
```

The count is the number of your functions that key **alone** unblocks — not
call sites. Clearing one blocker usually reveals the next, so the totals do not
add up and are not a forecast. Only single blockers are listed; a function
waiting on two keys needs both trusted at once and is not shown. The file is a
starting point, not an inventory.

**The counts are measured, not predicted.** bigo re-analyses your module with
each candidate priced and counts what actually became verifiable, so the number
beside a key is what asserting it delivers rather than what it looks like it
should. If that pass cannot run, the file says so at the top and the command
warns on stderr.

**Each count is an upper bound, and only a constant bound reaches it.** The
probe prices each candidate `O(1)`, which is the best case. If your honest
bound is in terms of an argument's size, the caller also needs that size to
resolve at the call site, and often it does not — measured on real
repositories, a truthful constant delivered 93% of its advertised count while a
truthful size-dependent one delivered 6%.

**Generated code counts.** Machine-written files in your module are scored like
any other, so a protobuf- or mock-heavy module can rank a key highly on the
strength of code nobody reads. Check what a key's callers actually are before
you spend reasoning on it.

Regenerating over an existing file needs `-force`, because your justifications
live there.

### Choosing a bound

Read the callee's implementation. That is the whole method, and it is not
advice — it is the rule this project pays for when it forgets:

> A cost is read from the implementation, never inherited from a neighbouring
> entry's precedent.

bigo has shipped seven wrong bounds, and the most recent came from pricing
`strings.Trim` by analogy with `strings.HasPrefix`. `HasPrefix` compares two
sequences and stops at the shorter; `Trim` tests membership of every rune of
its first argument in its second, so the lengths multiply. The analogy looked
sound and was wrong.

Write down why, next to the entry. An entry with no justification is a bound
nobody has reasoned about:

```
# (*cache.LRU).Get is O(1): a map lookup, then a move-to-front on a
# doubly-linked list. Nothing scans, and eviction is on the write path.
(*github.com/example/cache.LRU).Get O(1)
```

The example names a made-up dependency on purpose. A trust file is mostly for
third-party code, and a fabricated key cannot quietly become false when someone
else's implementation changes.

**This example used to be `os.Getenv O(1)`, justified as "it reads a pre-parsed
environ slice, no scan".** Both halves were wrong, and finding that out is the
best illustration this section has. On unix there *is* a scan — of the key, so
the honest bound is `O(len(key))`. On Windows `GetEnvironmentVariable` copies
the **value** into a buffer, so the cost is `O(len(value))`, and the value is
not an argument at all. bigo now refuses `os.Getenv` for exactly that reason and
`bigo trust init` no longer offers it. The worked example of reading an
implementation properly was itself a bound nobody had read properly.

### Using it

```
bigo json -trust bigo.trust          # the document
bigo -trust bigo.trust ./...         # the linter, budgets enforced
```

**A trusted bound is your claim, not bigo's inference.** Every influenced entry
in the report carries `provenance` (`assumed` on a trust entry's own target,
`assumption-tainted` downstream) and the document lists the whole trust surface,
so an auditor can find every verdict resting on an assertion. Trusted bounds
satisfy `//bigo:max` budgets — that is the point — and `bigo diff` reports a
verdict that moved because trust changed as `trust changed`, never as an
improvement.

**One consequence to know before you wire the CI gate.** `trust changed` is
neither a regression nor an improvement, so it never fails `-fail-on`. When the
trust surface differs between base and head, a verdict on code that rests on an
assertion is reported under that class **instead of** the class it would
otherwise have had — including a broken budget. A commit that edits the trust
file *and* makes trusted code asymptotically worse can therefore exit 0. Code
that does **not** depend on the changed trust still classifies normally, so an
ordinary regression is never masked by an unrelated trust edit. Review trust-file
changes as changes to what your gate checks, because that is what they are.

### What cannot be trusted

Interface methods and builtins cannot be named in a trust file at all; the
format has no key for them. `bigo trust init` never offers them, because it
filters on whether the blocker has a key rather than on a list someone has to
maintain.

### The honest limit

A trust entry is worth about one blocker. Functions sitting many blockers deep
are not reachable this way, and no file you are willing to write will change
that. This feature exists to make verdicts usable, not to raise a percentage.

### File format

```
# one entry per line: <key> <bound>; '#' starts a comment
os.Getenv O(1)
(*sync.Once).Do O(1)
example.com/dep.Work O(n) where n=len(xs)
```

Keys are package-qualified (receiver-qualified for methods, e.g.
`(*sync.Once).Do`, with generic instantiations resolved to their origin, e.g.
`(*sync/atomic.Pointer[T]).Load`); bounds use the `//bigo:cost` grammar, and
`where` bindings refer to the target's parameters by name.

Precedence is: in-source directive, then the curated cost table, then trust
entries, then inference. A shadowed entry — one a directive or a curated entry
already answers — warns. A malformed entry, an unparseable bound, or a key
matching nothing in the module is a hard error, never a skip (`bigo json`
validates keys module-wide; the analyzer driver validates syntax and
compilation only). Space bounds are not yet supported.

### `-assume`: the same mechanism, a different claim

`-assume` loads the identical format through the identical code path, and exists
for the what-if harness, which asks *what would this pricing buy?* rather than
asserting anything. Use `-trust` for claims you stand behind; passing both is an
error, because silently merging them would erase the distinction.

## Machine-readable reports

`bigo json` emits the full analysis as a single JSON document — every
function's inferred time (and, where budgeted, space) bound, three-valued
verdict, unverifiability causes, and the trust surface (`//bigo:cost` /
`//bigo:ignore` assertions in effect):

```sh
bigo json ./... > report.json
bigo json -C path/to/module -o report.json ./...
```

The document also carries a top-level `smells` array — every SM1–SM8 finding,
with rule, message, and position. It is top-level rather than per-function on
purpose: a smell is advisory and can never move a verdict, and the schema
mirrors that firewall. Functions carrying `//bigo:ignore` contribute no smells,
exactly as they emit no diagnostics.

Triage a codebase by rule with `jq`:

```sh
bigo json ./... | jq -r '.smells | group_by(.rule)[] | "\(.[0].rule): \(length)"'
bigo json ./... | jq -r '.smells[] | select(.rule=="SM8") | "\(.file):\(.line) \(.message)"'
```

The document format is versioned independently of bigo releases
(`schema_version`, currently 1.1.0) and specified normatively by
[`schema/report.schema.json`](schema/report.schema.json). Within a major
version, changes are additive-only and no field is ever reinterpreted —
consumers must ignore unknown fields. Verdicts never affect the exit code:
the report describes; enforcement belongs to tools built on it — `bigo diff`
and the GitHub Action below were its first consumers, and gate this
repository's own CI.

### Budget badge

`bigo badge` projects the report into a [shields.io endpoint badge](https://shields.io/badges/endpoint-badge) — an honest summary of the budgets a module declares, degrading from "all within" to name any `exceeds`, `invalid`, or `unverifiable` budget:

```sh
bigo badge ./... > badge.json                     # analyze and emit
bigo json ./... | bigo badge -i - > badge.json    # or project an existing report
```

Commit `badge.json` (or publish it as a CI artifact) and point shields.io at it:

```
https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/OWNER/REPO/main/badge.json
```

The badge claims only that the functions the module chose to budget are verified
within those budgets — a verified-contract statement about annotated API, like
coverage of tested code. It is not a repo grade, does not rank projects, and
never hides unverifiable budgets. Like `bigo json`, verdicts do not affect the
exit code.

### Complexity diff

`bigo diff` compares two reports and names what a change did to the module's
asymptotics — the CI-facing half of the schema:

```sh
git stash && bigo json ./... > base.json && git stash pop
bigo json ./... > head.json
bigo diff base.json head.json
```

It reports five classes, in severity order: a **budget break** (`within` →
`exceeds`), a **proven regression** (both sides proven, head asymptotically
worse — budget or not), a **new unverifiable** (a proven bound became `⊤`,
with the blocking cause named), a **new function already over budget**, and
**improvements** (`exceeds` → `within`, a tightened bound, `⊤` → proven).

Silence is the default and is deliberate: `⊤` → `⊤` reports nothing, unchanged
bounds report nothing, and a break that already existed in the base is not
blamed on this change. That is what makes diffing usable on real code where the
honest answer is often "not proven" — the noise cancels.

`-format markdown` renders a PR comment body; `-o` writes to a file; `-` reads
a document from stdin. Like `bigo json` and `bigo badge`, findings do not affect
the exit code unless you ask for it with `-fail-on break|regression`, which exits
3 on a violation — distinct from 1 (bigo failed) and 2 (usage).

Comparing reports from two different bigo versions is allowed but warns: a
bound may have changed because the engine improved rather than because the code
did. Comparing different modules, or across a schema major, is refused.

#### In CI

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0 # required: the base commit must be reachable
- uses: actions/setup-go@v5
  with:
    go-version-file: go.mod
- uses: RomanAgaltsev/bigo-action@v1
  with:
    fail-on: none # none (default) | break | regression
```

Posts a single PR comment and updates it in place on each push. `fail-on`
decides whether findings fail the job: `none` reports only, `break` fails on a
broken budget or a new function that arrives already over budget, `regression`
also fails on a proven asymptotic regression in unbudgeted code. Nothing fails
the job on a new unverifiable — losing visibility is worth telling you about,
but it is not a defect, and failing on it would just pressure you into avoiding
code bigo cannot yet see.

Report-only is the default on purpose: bigo's analysis surface is pre-stable
across minors, and a tool that breaks your build by surprise is a tool you
uninstall. Turn `fail-on` up once you trust it on your codebase.

The module must build for `bigo json` to analyze it. If the base commit does not
build — or predates your adoption of bigo — the Action says so and reports the
head side only, rather than failing.

## Install & run

```sh
go install github.com/RomanAgaltsev/bigo/cmd/bigo@latest

bigo ./...            # check budgets (CI-friendly exit code)
bigo -report ./...    # print the inferred bound of every analyzable function

bigo -C path/to/module ./...    # resolve ./... against another module
```

`-C dir` runs as if bigo had started in `dir`, so a CI job can analyze a module
without a `cd` shim. As with `go -C`, it must be the first flag. (`bigo json` and
`bigo badge` take their own `-C`.)

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
- **Non-constant multiplicity** (self-calls under a size loop),
  three-or-more-function recursion cycles, and per-level work whose cost depends
  on the recursion's *results* (merge sort's `merge(l, r)`, which would need
  relational length tracking). Two-function cycles are solved — see below.

### Mutual recursion

bigo solves **two-function cycles** `A → B → A` that thread a single size
measure, by composing the two per-edge steps into one virtual self-recurrence
and feeding it to the same subtractive/Master/Akra–Bazzi solvers. Even/odd
counters (`IsEven`/`IsOdd`, `O(n)`) and helper-mediated divide-and-conquer
(`WalkSum` splitting through a helper, `O(n log n)`) resolve. The same
well-foundedness proof applies per cycle: one member's guard suffices, and a
divisive cycle still requires the recursing side to prove the measure `>= 1`.

These stay unverifiable (⊤); annotate with `//bigo:cost` where a bound is known:

- **Three-or-more-function cycles** (recursive-descent parsers `expr → term →
  factor → expr`).
- **Mixed subtractive/divisive cycles** (one edge `n-1`, the other `n/2`).
- **Cycles through function values or interface methods** — the edge is not a
  static call, so the cycle is invisible.
- **Members that also self-recurse** (a larger SCC than the two-cycle).

Space budgets on mutual pairs remain unverifiable in this release.

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

## Function values

Calls through a function value used to be unverifiable across the board. bigo
now prices the common, statically resolvable shapes by summarizing a
higher-order function parametrically: a `Base` cost plus, per function-typed
parameter, an upper bound on how many times it is invoked. At a call site that
count is multiplied by the cost of the concrete argument.

What **resolves** today:

- **Static function arguments** — `Map(xs, double)` costs the invocation count
  times `double`'s own cost.
- **In-scope closures with an O(1) body** — the `sort.Slice` comparator that
  captures the slice only for O(1) index reads. A read-only captured slice's
  size is recovered even though Go boxes the capture (`sort.Slice(xs, less)` →
  `O(len(xs)·log(len(xs)))`).
- **Curated `sort`/`slices` callbacks** — `sort.Slice`, `sort.SliceStable`,
  `sort.Search`, `slices.SortFunc`, `slices.BinarySearchFunc`,
  `slices.ContainsFunc`/`IndexFunc`/`MaxFunc`/`MinFunc`/`CompactFunc`/`EqualFunc`,
  each priced as its documented-contract count × the callback cost.
- **Composition** — a helper that forwards its function parameter to another
  known-parametric helper composes the counts.
- **`range`-over-func over a stdlib producer** — `for v := range slices.Values(s)`
  (and `slices.All`/`Backward`, `maps.Keys`/`Values`/`All`) costs the producer's
  yield count times the loop body: an O(1) body over `slices.Values(s)` is
  `O(len(s))`.

The counting rule is a **whitelist**: a function parameter is priced only when
it is invoked directly or handed to another known-parametric callee. Every
other use — stored to a field or global, passed to an unknown/bodyless callee,
launched in a goroutine, captured then mutated, or read from a struct field or
channel — forces its count to `⊤`, so an invocation bound is never
under-counted.

What still stays `⊤` (annotate the callee with `//bigo:cost`, or trust it): a
closure whose body cost depends on a captured size (product bounds are
deferred), a closure created in one function and consumed in another, a func
value from a struct field or channel, goroutine-invoked callbacks, and
`range`-over-func over a user-defined or recursive iterator (only the curated
stdlib producers above resolve).

## What bigo does not count (yet)

Each can only *miss* a violation, never invent one:

- `append` is amortized O(1) (including `append(a, b...)`), `make` is O(1).
- String concatenation and comparison are O(1) per operation.
- Map index/assign/delete are O(1) *time*. On the **space** axis an assign is
  charged its amortized O(1) allocation, so a map grown in a loop is O(trips)
  heap — a map sized to its input no longer looks allocation-free.
- Self-recursion over a size measure is solved (see [Recursion](#recursion));
  the statically resolvable function-value shapes are priced (see
  [Function values](#function-values)). Interface calls without `//bigo:cost`,
  out-of-scope closures, capture-sized closures, and `range`-over-func
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

## Smells

In addition to the three-valued budget verdicts, bigo emits **advisory
complexity smells** — patterns that are *provably* the shape of a known
inefficiency, reported with the prefix `smell(SMn):`. Smells are firewalled
from verdicts: they never affect a budget's within/exceeds/unverifiable result,
never read a budget, and are never consulted by the cost engine. They catch
the patterns bigo's allocation-blind cost model deliberately does not charge
(string concatenation), plus ones no Go linter can name (exponential
recursion).

A smell fires only on a **proved** SSA pattern — the analogue of bigo's ⊤ rule:
when the detector cannot prove the pattern, it stays silent. Diagnostics carry
the fixed `smell(SMn):` prefix so golangci-lint users can filter on the class.

| Rule | Fires on | Headline no-fire |
|---|---|---|
| **SM1** | string built by `+=` (or `fmt.Sprintf` self-accumulation) in a data-dependent loop | `strings.Builder`; constant-trip loop |
| **SM2** | repeated `slices.Contains`/`Index` over a parameter slice with a loop-varying needle | loop-invariant needle; rebuilt scan target |
| **SM3** | `append` into a zero-capacity slice bounded by a resolvable loop with a *named* bound | `make([]T, 0, n)` with capacity given; a constant loop bound (there is no size to name) |
| **SM4** | `regexp.Compile`/`MustCompile` inside any loop, from a pattern invariant across the innermost enclosing loop | pattern built per iteration (nothing to hoist); compile already hoisted before the loop |
| **SM5** | sorting inside a data-dependent loop, on a slice that is stable across iterations and not written to inside the loop | slice rebuilt or mutated per iteration (the sort is necessary work); `sort.Sort`/`sort.Stable` (a `sort.Interface` exposes no slice to check); constant-trip loop; sort outside any loop |
| **SM6** | `make(map[K]V)` without a size hint, grown in a resolvable loop with a *named* bound | `make(map[K]V, n)` with a hint; a constant loop bound |
| **SM7** | a redundant second lookup the first already answered (map comma-ok then plain; `slices.Contains` then `slices.Index`) | a single lookup; a map mutated between them |
| **SM8** | provably exponential recursion (Θ(aⁿ), a ≥ 2 — naive Fibonacci) | linear countdown (a=1); divisive binary search; unguarded recursion; **memoized recursion** (a comma-ok cache hit dominating the self-calls — O(n), not exponential) |

SM8 is the differentiator: it is powered by the recurrence solver's termination
proof and branching-factor analysis, so it can positively identify the Θ(aⁿ)
family — a diagnostic no other Go linter can make.

### The `-smells` flag

```sh
bigo -smells=none ./...     # budgets only, no smells
bigo -smells=SM1,SM8 ./...  # just two rules
bigo -smells=all ./...      # all rules (the default)
```

`//bigo:ignore` suppresses smells on a function just as it suppresses budget
analysis. `//nolint:bigo` in golangci-lint suppresses individual diagnostics.

### Filtering smells out of a budgets-only pipeline

Teams wanting budget verdicts without the advisory smells can filter on the
`smell(` prefix:

```yaml
# .golangci.yml — exclude the smell class, keep budget diagnostics
issues:
  exclude-rules:
    - path: "\.go$"
      text: "^smell\\(SM\\d\\):"
      linters:
        - bigo
```

The fire counts per rule are tracked as a drift alarm in `metrics/METRICS.md`
("Smell fires") — not coverage, but a change in a rule's corpus count is a
behavior change that must be deliberate.

## Canonical algorithm corpus (the oracle)

`corpus/testdata/src/` holds ~55 textbook algorithms whose worst-case bounds
are known from the literature, pinned in-source:

```go
//oracle:time O(n^2) where n=len(s)
//oracle:space O(1) where n=len(s)
//oracle:source CLRS §2.1
func InsertionSort(s []int) { … }
```

`internal/oracle` runs bigo's inference over them **unaided** (no `//bigo:`
directives) and compares by bound domination: an emitted bound that does not
dominate its pin is a wrong bound and **fails the build** — the prime
directive, mechanically enforced. Sound results land in the committed golden
([corpus/CORPUS.md](corpus/CORPUS.md), regenerate with `task corpus`):
`exact` (matches the literature), `loose` (sound, imprecise — a graduation
target), or `top` (unverifiable — these rows are the evidence base for the
annotate-or-trust recipes below). Algorithms that cannot be soundly pinned
are listed in [corpus/EXCLUSIONS.md](corpus/EXCLUSIONS.md) with reasons —
notably pointer-backed structures (BST/list/trie), whose sizes are not
parameter sizes: annotate those with `//bigo:cost` at the call boundary or
budget the caller.

The corpus is **not** a coverage metric. Read composition, not a percentage.

### Reach: the real-world survey

The corpus answers "did we get the literature's answer?" and `metrics/` answers
"did anything drift?". Neither answers "how much real Go can bigo bound, and what
stops it?" — so a third instrument does.

[survey/SURVEY.md](survey/SURVEY.md) records a run of the shipped analyzer over
external repositories listed in `survey/targets.json`, reporting coverage over
**first-party functions only** (dependencies are filtered by module path) plus a
ranked histogram of what blocked the rest. It leads with three numbers rather
than one, because a single coverage figure averages populations that behave
differently:

- **coverage** — how much of the module bigo bounds;
- **the near frontier** — ⊤ functions within two distinct blockers of a bound,
  reported beside a `ceiling_pct` that is an upper bound and never a forecast.
  Measured 2026-07-20: 29% of ⊤ functions sit **one** blocker from a bound
  while 25% sit ten or more, so incremental work reaches one population and not
  the other;
- **hand-written versus generated** — generated code is first-party by module
  path and is real code, but nobody hand-tunes it and its unverifiability is
  usually the *correct* answer. Every ranking table counts hand-written code
  only.

Work is ranked by the **graduation-count** table — functions whose *only*
blocker is one entry — not by the site histogram beside it. Those disagree, and
ranking by sites has twice produced directions that measured out at nothing.

Unlike the other two it is a **manual measurement, not a golden test** — its
inputs are repositories on one machine at whatever commit they sit, and its
numbers are expected to move. `task survey` regenerates it; CI never runs it.

### Removability: the what-if harness

The fourth instrument answers the question the other three cannot: **what
would actually graduate if a given function were priced?** `task whatif --
candidates.json` runs the real engine over the survey targets once per
candidate assumption set and reports, per candidate, the exact functions that
go ⊤ → bounded with assumption provenance — never a graph estimate, which
measured 50 functions optimistic on this codebase's own frontier probe. A
result is **withheld entirely** if any graduation lacks provenance, any bound
is lost, or the population mismatches: an unattributed count invalidated two
probe harnesses only after their headlines looked plausible, so here the
counter is a gate, not a footnote. Output lands in `survey/WHATIF.md` +
`survey/whatif.json`; like the survey it is manual, machine-local, and never
a golden.

## Status & versioning

Complete: intraprocedural engine, cost tables, acyclic interprocedural
summaries, generics at instantiation, golangci-lint plugin, size-measure
recurrence solving (subtractive / Master / Akra–Bazzi), space complexity
(`//bigo:space`), mutual recursion, function-value costs, code-smell rules,
and the ecosystem commands (`bigo json`, `bigo badge`, `bigo diff`).
The **analysis surface is pre-stable**: verdicts may change between minor
versions as inference improves.

Permanently unverifiable by design (annotate with `//bigo:cost` or
`//bigo:ignore`): unannotated interface dispatch, worklist loops, pointer-chasing
recursion, concurrency, and `reflect`/`unsafe`.

## License

MIT
