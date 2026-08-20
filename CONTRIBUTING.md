# Contributing to bigo

- Run `task ci` before pushing (tidy + lint + race tests). `task lint:custom`
  additionally runs bigo on itself via the custom golangci-lint binary.
- PR titles are Conventional Commits (`feat:`, `fix:`, `chore:`, ...) — the
  `pr-title` gate enforces this, and release-please derives versions from it.
- All changes land via squash-merged PRs; the `lint-success`, `test-success`,
  and `security-success` gates must pass.
- **Soundness rule:** bigo must never emit a wrong bound. When the analysis
  cannot prove a bound, return `bound.Top()` (unverifiable). A false negative
  is acceptable; a false positive or wrong bound is a release-blocking bug.
- New analysis behavior needs an `analyzer/testdata/src/` corpus entry
  asserting the verdict — including the unverifiable ones.

## Measuring bigo's behaviour

`bin/` is gitignored, and its contents are a **cache, not a build**. Run
`task build` before drawing any conclusion from bigo's output.

A stale binary answers every flag and produces plausible, internally
consistent, bisectable results — nothing about its behaviour announces that it
is old. In August 2026 a six-week-old copy produced an entire design document
whose headline findings all had to be retracted, and **two rounds of bisection
refined the wrong finding rather than exposing it**, because every probe reused
the same artifact.

So treat "several probes agree" as no evidence at all when they share one
binary: vary the artifact, not just the input. If a result is surprising *and*
bigo is supposed to be good at that thing, suspect the artifact first.
