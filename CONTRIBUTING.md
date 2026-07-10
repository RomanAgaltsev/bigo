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
