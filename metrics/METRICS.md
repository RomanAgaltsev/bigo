# bigo corpus coverage

GENERATED — do not edit; regenerate with `task metrics`.

**Coverage: 57.6%** — 106 of 184 corpus functions bounded.

## Per package

| Package | Functions | Bounded | Unverifiable |
|---|---|---|---|
| budget | 9 | 7 | 2 |
| concurrent | 5 | 4 | 1 |
| costignore | 10 | 4 | 6 |
| edge | 11 | 5 | 6 |
| fieldsize | 13 | 8 | 5 |
| funcvalue | 20 | 8 | 12 |
| interproc | 5 | 5 | 0 |
| iterator | 6 | 4 | 2 |
| multivar | 3 | 3 | 0 |
| mutual | 20 | 7 | 13 |
| negative | 3 | 0 | 3 |
| positive | 4 | 4 | 0 |
| recursion | 16 | 9 | 7 |
| smells | 24 | 13 | 11 |
| smoke | 2 | 1 | 1 |
| space | 8 | 7 | 1 |
| structures/array | 7 | 7 | 0 |
| structures/graph | 3 | 1 | 2 |
| structures/hashmap | 3 | 3 | 0 |
| structures/heap | 4 | 3 | 1 |
| structures/list | 3 | 1 | 2 |
| structures/stringops | 2 | 2 | 0 |
| structures/tree | 3 | 0 | 3 |

## Unverifiable by cause

| Cause | Count |
|---|---|
| call | 51 |
| go | 3 |
| irreducible | 1 |
| loop | 15 |
| nobody | 8 |

The cause histogram is the Phase-2 prioritization signal: the biggest
bucket is the next feature.

## Smell fires (drift alarm)

Not coverage. A change in a rule's corpus fire count is a behavior change and must be deliberate.

| Rule | Corpus fires |
|---|---|
| SM1 | 4 |
| SM2 | 1 |
| SM3 | 1 |
| SM4 | 1 |
| SM5 | 2 |
| SM6 | 1 |
| SM7 | 1 |
| SM8 | 2 |
