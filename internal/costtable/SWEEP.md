# The one-argument sweep — 2026-08-12

Every entry in `stdlib` that names **exactly one** argument, checked against its
signature and its implementation, asking one question:

> Can a **second** argument multiply this cost?

**Why this file exists.** v1.38.1 fixed the seventh wrong bound (`strings.Trim`)
and named its class — *the cost depends on something other than what the entry
is sized by*. The class then acquired five more recorded instances, every one
found individually while implementing something else, and the table was never
walked. The 2026-08-12 review found the eighth wrong bound sitting in it. **A
class that is named but not swept is a backlog nobody opened.**

The fine entries are recorded too. A list of only the defects is a list someone
has to re-derive; this one is a checked inventory the next reader inherits.

**Standing rule this earns:** when a review names a defect CLASS, the fix
closes the instance **and files the sweep**. When a new one-argument entry is
added, add its row here.

## What changed as a result

| Entry | Was | Now | Why |
|---|---|---|---|
| `strings.Join` | `O(len(elems))` | `productUnlessConst(0, 1)` | writes `sep` `len(elems)-1` times — **8th wrong bound** |
| `strings.Replace` | `O(len(s))` | `productUnlessConst(0, 2)` | writes `new` once per replacement, count ≤ `len(s)` — **8th wrong bound** |
| `strings.ReplaceAll` | `O(len(s))` | `productUnlessConst(0, 2)` | as `Replace` |
| `strings.Index`, `Contains`, `Count`, `LastIndex`, `Cut`, `Split`, `SplitN` | `O(len(s))` | `searchCost` | needle > `bytealg.MaxLen` reaches `IndexRabinKarp`, whose per-collision verify is `O(len(needle))` |
| `bytes.Index`, `Contains`, `Count` | `O(len(b))` | `searchCost` | same implementation family |

Everything else in the sweep was checked and left alone.

## A — comparison entries: stop at the shorter operand

`O(min) ≤ O(len(arg0))`, so sizing by argument 0 is a valid upper bound. This is
the pre-existing `slices.Equal` / `strings.HasPrefix` precedent and it is
correct — it is also the precedent `Trim` was wrongly priced by, so the line
matters: these entries **compare** two sequences, `Trim` **searches** one in the
other.

| Entry | Second argument | Verdict |
|---|---|---|
| `strings.HasPrefix` | `prefix` | fine — `O(len(prefix)) ≤ O(len(s))` |
| `strings.HasSuffix` | `suffix` | fine — same |
| `strings.TrimPrefix` | `prefix` | fine — `HasPrefix` plus a slice expression |
| `strings.TrimSuffix` | `suffix` | fine — same |
| `strings.EqualFold` | `t` | fine — single pass, stops at the shorter |
| `strings.Compare` | `b` | fine — `bytealg.CompareString` stops at `min` |
| `bytes.Equal` | `b` | fine |
| `bytes.HasPrefix` | `prefix` | fine |
| `bytes.HasSuffix` | `suffix` | fine |
| `slices.Equal` | `s2` | fine — subject to the unit-element axiom |

## B — single pass over argument 0, no second size

| Entry | Verdict |
|---|---|
| `strings.Fields` | fine — one scan, result slice bounded by `len(s)` |
| `strings.ToLower` / `ToUpper` / `Title` | fine — per-rune case mapping is a binary search of a **fixed** table, a machine constant |
| `strings.TrimSpace` | fine — no cutset argument; the space set is compiled in |
| `strings.IndexByte` / `bytes.IndexByte` | fine — a **single byte** has no length to multiply. Deliberately excluded from `searchCost`; widening it would be a pure capability loss |
| `strconv.Atoi` / `ParseInt` / `ParseUint` / `ParseBool` | fine — linear in the string, the base and bit-size arguments are numeric |
| `net/http.CanonicalHeaderKey` | fine — a constant number of linear passes plus a map lookup |
| `net/netip.ParsePrefix` / `MustParsePrefix` | fine — split at the last `/`, then two linear parses |
| `slices.Reverse` | fine — `len(s)/2` swaps, each fixed-width for a given element type |
| `maps.Clone` | fine — one entry per key; subject to the unit-element axiom |
| `maps.Copy` (argument **1**) | fine — ranges over `src`; sizing it by the destination would be a different wrong bound |

## C — the substring-search family: DECIDED, changed

`strings.Index`, `Contains`, `Count`, `LastIndex`, `Cut`, `Split`, `SplitN`,
`bytes.Index`, `bytes.Contains`, `bytes.Count`.

Reading `internal/stringslite/strings.go` and `internal/bytealg`:

- a needle at or below `bytealg.MaxLen` (31 or 63 bytes depending on CPU) is
  brute-forced with a comparison bounded by that **machine constant** — the same
  licence GOMAXPROCS gets in `(*sync.Pool).Put` and the word width gets in
  `bits.Div64`. `O(len(s))` is sound here;
- a **longer** needle falls through to `IndexRabinKarp`, which verifies every
  hash match with a full compare. `PrimeRK` is a compile-time constant, so
  collisions are constructible and the worst case is `len(s) · len(needle)`.

**Decision: price the product, with a constant arm.** The 2026-08-12 review
declined to call this a break because the worst case is adversarial and remote.
That is true, and it is not the test this project applies — the prime directive
is a **worst-case** upper bound and ⊤ is always safe. "It holds for typical
inputs" is precisely the reasoning that produced `Trim`.

Cost of deciding this way, checked rather than assumed:

- a **literal** needle — `strings.Contains(s, "://")`, `strings.Split(s, ",")` —
  takes the constant arm and keeps the linear bound. That is the overwhelming
  majority of real calls, and it is pinned as a positive control;
- a **variable** needle still gets a **bound**, not ⊤, wherever both extents
  resolve. This narrows bounds; it does not lose verdicts except where the
  needle's size is unresolvable;
- the canonical corpus and the analyzer fixtures call none of these, so corpus,
  metrics and report goldens are **byte-identical**. **The survey will move and
  that movement is unmeasured** — it must be re-run and the delta explained.

## Additions — blind-repo lane 2, 2026-08-12

The rule from this file's header, applied on its first occasion: a new
one-argument entry owes a row here. Each was checked for a second argument that
could multiply its cost.

**Note the argument index on every method.** `ssa.CallCommon.Args` carries the
receiver at index 0, so a method's first declared parameter is index **1**.
Sizing `(net/http.Header).Get` by index 0 would name the header map instead of
the key — a different quantity that compiles fine and reads as correct. Pinned
by a test that asserts the bound names the key.

| Entry | Sized by | Second argument | Verdict |
|---|---|---|---|
| `(net/http.Header).Get` | arg 1, the key | — | fine — canonicalises the key, one map lookup; the number of headers does not enter |
| `(net/http.Header).Set` | arg 1, the key | `value` | fine — the value is stored, never scanned |
| `(*encoding/base64.Encoding).EncodeToString` | arg 1, the source | — | fine — one pass |
| `(*encoding/base64.Encoding).DecodeString` | arg 1, the string | — | fine — one pass |
| `slices.Clone` | arg 0 | — | fine — one copy; unit-element axiom applies |
| `bytes.Compare` | arg 0 | `b` | fine — stops at `min`, and `[]byte` elements are fixed-width so the axiom is not even in play |
| `strings.IndexRune` | arg 0 | a single rune | fine — a rune has no length to multiply, the `IndexByte` case |
| `strconv.ParseFloat` | arg 0 | bit size, numeric | fine |
| `time.ParseDuration` | arg 0 | — | fine |
| `path/filepath.Clean` | arg 0 | — | fine |
| `net/url.PathEscape` | arg 0 | — | fine |

Constant entries added in the same lane — `math` ×61, `sync.NewCond`,
`time.Sleep` — name no argument at all and so cannot be members of this class.
The `math` family argument and its two declared refusals are documented at the
entries.

## D — sorting

| Entry | Verdict |
|---|---|
| `slices.Sort`, `sort.Ints`, `sort.Float64s` | fine — fixed-width elements, `O(n log n)` exact |
| `sort.Strings`, `slices.Sort` over strings | fine **under the unit-element axiom** — each comparison is `O(min len)`, which the axiom charges as unit. See E |
| `slices.BinarySearch` | fine — `O(log n)` comparisons, same axiom |
| `strings.Repeat` | already `prodOf(0, 1)` — the entry that shows the table always knew this shape |

## E — element content: DECIDED, unchanged, now written down

`slices.Contains`, `slices.Index`, `slices.Max`, `slices.Min`, `slices.Equal`,
`sort.Strings`, `slices.Sort`, `slices.BinarySearch`, `maps.Clone`, `maps.Copy`.

For a `[]string`, each element comparison costs `O(len(element))`, so the true
cost of `slices.Contains(xs, target)` is `O(len(xs) · len(target))` — and unlike
the content sum, `target` **is** nameable, so the product **is** expressible.

**Decision: keep unit cost per element, and state it as an axiom.** It now lives
in the package doc under *the unit-element axiom*. Three reasons:

1. A size variable in bigo's grammar is a **count**. Σ|xs[i]| has no name, so
   the only sound alternative is ⊤ for every content-dependent element
   operation.
2. That alternative makes sorting a `[]string` ⊤ and **contradicts the canonical
   corpus**, which pins sorting at `O(n log n)` whatever it sorts. The oracle
   would go red on entries the literature agrees with.
3. A type-directed cost model — unit for fixed-width elements, product for
   variable-width — is the only way to have both, and it is a real engine
   change, not a table edit. Pricing `slices.Contains` as a product
   *unconditionally* would turn the common `[]int` case into ⊤, because `len`
   of an `int` does not resolve. **That would be a capability loss sold as a
   soundness fix.**

**The boundary is what makes the axiom honest**, and it is the same line the
sweep drew everywhere else: the axiom covers an **element of the collection
being sized**; it does not cover a **separate, nameable argument**. `Trim`'s
cutset, `Join`'s separator, `Replace`'s replacement and the search family's
needle are all the second kind.

**Recorded as the open question this decision does not close:** a type-directed
element cost is the only route to pricing `slices.Contains` over a `[]string`
honestly *and* keeping `[]int` bounded. Nobody has measured what it would buy.
It needs a probe with pre-registered bars before it needs a spec.
