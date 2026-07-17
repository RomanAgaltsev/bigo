# Canonical corpus — exclusion register

Algorithms considered and deliberately kept out, with reasons (spec §3.5).
This is the anti-drift mirror of the pin discipline: what the corpus does
NOT claim is written down too. An entry leaves this list only when its
reason stops holding (e.g. the bound algebra grows a new form).

| Algorithm | Reason |
|---|---|
| Naive recursive Fibonacci | true bound O(φ^n) — exponentials are inexpressible in the poly-log bound algebra |
| Karatsuba multiplication | true bound O(n^1.585) — fractional exponents inexpressible |
| Tower of Hanoi | O(2^n) — exponential, inexpressible |
| BST insert/search, linked-list ops, trie ops (pointer-backed) | structure size is not a parameter size: canonical variables are len/cap/value of parameters, so no sound pin can be stated (spec §3.3.1). This is the annotate-or-trust family — see the README recipes |
| DSU find/union WITH path compression | amortized inverse-Ackermann bound — amortized claims are inexpressible; the un-amortized worst case IS in the corpus (structures.DSUFind) |
| Grid flood fill / anything n×m over a nested slice | the inner dimension len(g[0]) is not bindable in a where-clause; only parameter sizes are |
| Quickselect | expected-case O(n) is an average-case claim; its worst case O(n^2) duplicates QuickSort's shape |
| Sieve of Eratosthenes at its true bound | O(n log log n) inexpressible — INCLUDED but with a stated conservative pin O(n log n) (numeric.Sieve), recorded here for honesty |
| Trial division at its true bound | O(√n) inexpressible — INCLUDED with a stated conservative pin O(n) (numeric.TrialDivision) |
