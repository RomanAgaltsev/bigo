# bigo contribution lane — triage queue

GENERATED — do not edit; regenerate with `task contrib-scan`.

**A MANUAL measurement, not a golden test.** No test asserts its contents and
CI never runs it. Targets are repositories on one machine at whatever commit
they happen to sit; compare runs only via the per-target commit below.

Run 2026-08-19 with bigo 1.47.0.

**Sample: 40 findings, at most 8 per rule and 6 per target**, drawn in target order then file, line, rule. The rule was registered in `docs/bigo/investigations/2026-08-18-contribution-lane-thresholds.md` before this scan ran, and is implemented in `survey.Sample` so the two cannot drift. The per-target cap is Amendment 1, made before any verdict was assigned.

Population: first-party, hand-written. Generated code is excluded — nobody
hand-tunes it, so a finding there is not a contribution.

## Targets scanned

| Target | Module | Commit | Findings |
|---|---|---|---|
| caddy | github.com/caddyserver/caddy/v2 | 0e8eb41b | 24 |
| nats-server | github.com/nats-io/nats-server/v2 | 2e5f51f31 | 25 |
| hugo | github.com/gohugoio/hugo | 89b8c3220 | 66 |
| prometheus | github.com/prometheus/prometheus | a0524eeca | 62 |
| grpc-go | google.golang.org/grpc | 2fd426d0 | 46 |
| pgx | github.com/jackc/pgx/v5 | 0a977a6 | 4 |
| goldmark | github.com/yuin/goldmark | 50ba9fc | 4 |
| cel-go | github.com/google/cel-go | 646511d | 6 |
| delve | github.com/go-delve/delve | 8fc4acbd | 14 |
| chi | github.com/go-chi/chi/v5 | 3b17157 | 2 |

## Findings by rule (whole population, not the sample)

| Rule | Findings |
|---|---|
| SM3 | 121 |
| SM6 | 115 |
| SM1 | 9 |
| SM2 | 6 |
| SM4 | 2 |

## The sample

Verdict column is filled by hand during triage: `sendable`, `not-sendable`,
or `FP`. Read the target's own `.golangci.yml` before judging anything
sendable — a finding its CI already declines to enforce is not-sendable
however correct it is.

| # | Target | Rule | Position | Message | Verdict |
|---|---|---|---|---|---|
| 1 | caddy | SM6 | `admin.go:301` | map grown in a loop bounded by O(len(admin.Origins)) without a size hint; preallocate with make(map[K]V, O(len(admin.Origins))) | |
| 2 | caddy | SM1 | `caddyconfig/caddyfile/lexer.go:317` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 3 | caddy | SM3 | `caddyconfig/caddyfile/parse.go:798` | append in a loop bounded by O(len(sb.Segments)) on a zero-capacity slice; preallocate with make(…, 0, O(len(sb.Segments))) | |
| 4 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:83` | map grown in a loop bounded by O(len(originalServerBlocks)) without a size hint; preallocate with make(map[K]V, O(len(originalServerBlocks))) | |
| 5 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:96` | map grown in a loop bounded by O(len(originalServerBlocks)) without a size hint; preallocate with make(map[K]V, O(len(originalServerBlocks))) | |
| 6 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:118` | map grown in a loop bounded by O(len(originalServerBlocks)) without a size hint; preallocate with make(map[K]V, O(len(originalServerBlocks))) | |
| 7 | nats-server | SM6 | `server/accounts.go:708` | map grown in a loop bounded by O(len(dests)) without a size hint; preallocate with make(map[K]V, O(len(dests))) | |
| 8 | nats-server | SM3 | `server/certidp/certidp.go:196` | append in a loop bounded by O(len(uris)) on a zero-capacity slice; preallocate with make(…, 0, O(len(uris))) | |
| 9 | nats-server | SM3 | `server/certstore/certstore_windows.go:179` | append in a loop bounded by O(len(caCertsMatch)) on a zero-capacity slice; preallocate with make(…, 0, O(len(caCertsMatch))) | |
| 10 | nats-server | SM3 | `server/client.go:6713` | append in a loop bounded by O(len(cts)) on a zero-capacity slice; preallocate with make(…, 0, O(len(cts))) | |
| 11 | nats-server | SM1 | `server/errors.go:319` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 12 | nats-server | SM1 | `server/errors.go:322` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 13 | hugo | SM3 | `codegen/methods.go:357` | append in a loop bounded by O(len(m)) on a zero-capacity slice; preallocate with make(…, 0, O(len(m))) | |
| 14 | hugo | SM3 | `codegen/methods.go:402` | append in a loop bounded by O(len(m)) on a zero-capacity slice; preallocate with make(…, 0, O(len(m))) | |
| 15 | hugo | SM3 | `codegen/methods.go:449` | append in a loop bounded by O(len(f)) on a zero-capacity slice; preallocate with make(…, 0, O(len(f))) | |
| 16 | hugo | SM6 | `codegen/methods.go:510` | map grown in a loop bounded by O(len(s)) without a size hint; preallocate with make(map[K]V, O(len(s))) | |
| 17 | hugo | SM3 | `codegen/methods.go:516` | append in a loop bounded by O(len(s)) on a zero-capacity slice; preallocate with make(…, 0, O(len(s))) | |
| 18 | hugo | SM6 | `commands/convert.go:206` | map grown in a loop bounded by O(len(pagesBackedByFile)) without a size hint; preallocate with make(map[K]V, O(len(pagesBackedByFile))) | |
| 19 | prometheus | SM6 | `cmd/promtool/analyze.go:95` | map grown in a loop bounded by O(len(matchers)) without a size hint; preallocate with make(map[K]V, O(len(matchers))) | |
| 20 | prometheus | SM2 | `promql/promqltest/test.go:1191` | repeated linear scan over the same slice (quadratic); build a map/set once before the loop | |
| 21 | prometheus | SM2 | `promql/promqltest/test.go:1199` | repeated linear scan over the same slice (quadratic); build a map/set once before the loop | |
| 22 | prometheus | SM1 | `scrape/scrape.go:628` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 23 | grpc-go | SM1 | `internal/resolver/dns/dns_resolver.go:303` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 24 | grpc-go | SM1 | `xds/internal/xdsclient/xdsresource/matcher.go:129` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 25 | goldmark | SM2 | `testutil/testutil.go:167` | repeated linear scan over the same slice (quadratic); build a map/set once before the loop | |
| 26 | cel-go | SM1 | `common/error.go:61` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
