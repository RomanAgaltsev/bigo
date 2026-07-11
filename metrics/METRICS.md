# bigo corpus coverage

GENERATED — do not edit; regenerate with `task metrics`.

**Coverage: 56.0%** — 42 of 75 corpus functions bounded.

## Per package

| Package | Functions | Bounded | Unverifiable |
|---|---|---|---|
| budget | 9 | 7 | 2 |
| concurrent | 5 | 4 | 1 |
| costignore | 10 | 4 | 6 |
| edge | 11 | 4 | 7 |
| interproc | 5 | 5 | 0 |
| multivar | 3 | 3 | 0 |
| negative | 3 | 0 | 3 |
| positive | 4 | 4 | 0 |
| smoke | 1 | 1 | 0 |
| structures/array | 7 | 3 | 4 |
| structures/graph | 3 | 1 | 2 |
| structures/hashmap | 3 | 2 | 1 |
| structures/heap | 3 | 1 | 2 |
| structures/list | 3 | 1 | 2 |
| structures/stringops | 2 | 2 | 0 |
| structures/tree | 3 | 0 | 3 |

## Unverifiable by cause

| Cause | Count |
|---|---|
| call | 7 |
| go | 2 |
| irreducible | 1 |
| loop | 17 |
| nobody | 6 |

The cause histogram is the Phase-2 prioritization signal: the biggest
bucket is the next feature.
