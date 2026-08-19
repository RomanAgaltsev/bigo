# bigo real-world survey

GENERATED — do not edit; regenerate with `task survey`.

**This is a MANUAL measurement, not a golden test.** No test asserts its
contents and CI never runs it. Its targets are repositories that exist on one
machine at whatever commit they happen to sit, so these numbers are a record
of one run — compare across runs only via the per-target commit below.

Run 2026-08-19 with bigo 1.46.0 — the last RELEASED version at run time. A run made before its own release stamps the previous tag, so compare runs by the per-target commit below, never by this line.

**Aggregate: 32.9%** — 11021 of 33504 first-party functions bounded.

**Hand-written: 31.6%** — 9229 of 29214 functions bounded, with 4290 generated functions excluded.

Generated code is first-party by module path and is real code, but nobody
hand-tunes it and its unverifiability is usually the CORRECT answer — the
2026-07-21 `(*sync.Once).Do` probe measured 239 of that class's 326
sole-blocker functions as generated protobuf whose verdict is right.
**The aggregate above is kept unrebased** so it stays comparable with the
2026-07-20/21 probes, which pin their population to it.

**Hand-written near frontier: 8217 of 19985 (41.1%), ceiling 59.7%.**

**Near frontier: 9834 of 22483 unverifiable functions (43.7%) sit within 2 distinct blockers of a bound.** Clearing all of them would put coverage at **62.2%** — an UPPER BOUND, not a forecast: clearing a blocker for one function need not clear it for another. Two 2026-07-20 probes measured that gap directly (`fmt`: 744 sole-blocker functions, 298 actually priceable; function values: 573, zero reachable).

## Per target

| Target | Module | Commit | Functions | Bounded | Coverage | Generated | Hand | Hand cov | Near | Ceiling |
|---|---|---|---|---|---|---|---|---|---|---|
| grpc-go | google.golang.org/grpc | 2fd426d0 | 5467 | 1973 | 36.1% | 1902 | 3565 | 30.0% | 1971 | 72.1% |
| caddy | github.com/caddyserver/caddy/v2 | 0e8eb41b | 1963 | 473 | 24.1% | 0 | 1963 | 24.1% | 471 | 48.1% |
| prometheus | github.com/prometheus/prometheus | a0524eeca | 5859 | 1971 | 33.6% | 776 | 5083 | 31.8% | 1755 | 63.6% |
| etcd | go.etcd.io/etcd/v3 | 22b4192b9 | 98 | 9 | 9.2% | 0 | 98 | 9.2% | 40 | 50.0% |
| delve | github.com/go-delve/delve | 8fc4acbd | 2793 | 734 | 26.3% | 28 | 2765 | 26.3% | 722 | 52.1% |
| chi | github.com/go-chi/chi/v5 | 3b17157 | 180 | 66 | 36.7% | 0 | 180 | 36.7% | 55 | 67.2% |
| goldmark | github.com/yuin/goldmark | 50ba9fc | 795 | 442 | 55.6% | 0 | 795 | 55.6% | 130 | 71.9% |
| pgx | github.com/jackc/pgx/v5 | 0a977a6 | 2099 | 784 | 37.4% | 110 | 1989 | 37.6% | 660 | 68.8% |
| cel-go | github.com/google/cel-go | 646511d | 3586 | 1545 | 43.1% | 937 | 2649 | 40.6% | 1115 | 74.2% |
| expr | github.com/expr-lang/expr | 4b31df3 | 1286 | 232 | 18.0% | 515 | 771 | 30.1% | 214 | 34.7% |
| nats-server | github.com/nats-io/nats-server/v2 | 2e5f51f31 | 4000 | 1007 | 25.2% | 0 | 4000 | 25.2% | 1084 | 52.3% |
| hugo | github.com/gohugoio/hugo | 89b8c3220 | 5378 | 1785 | 33.2% | 22 | 5356 | 32.9% | 1617 | 63.3% |

## Distance to bound

How many DISTINCT leaf blockers stand between an unverifiable function and a
bound, walking through propagation. This is why a single headline coverage
number is misleading: it averages a near frontier that incremental work can
reach against a deep tail that no achievable engine work will.

| Blockers | Functions | Share |
|---|---|---|
| 0 | 31 | 0.1% |
| 1 | 6583 | 29.3% |
| 2 | 3220 | 14.3% |
| 3 | 1988 | 8.8% |
| 4 | 1665 | 7.4% |
| 5 | 1018 | 4.5% |
| 6 | 749 | 3.3% |
| 7 | 684 | 3.0% |
| 8 | 492 | 2.2% |
| 9 | 368 | 1.6% |
| 10+ | 5685 | 25.3% |

## Unverifiable by cause kind

| Cause | Count |
|---|---|
| call | 92332 |
| loop | 43226 |
| defer | 828 |
| go | 335 |
| nobody | 3 |

Compare with `corpus/CORPUS.md`: the canonical corpus and real code do not
agree on this ranking, and real code is the one that reflects adoption.

Population: hand-written code only.

## Top 30 blockers by GRADUATION count

**This table is the deliverable.** It counts functions whose ONLY blocker is
each entry — the number that would actually graduate if it were cleared.

A class here is one cause detail verbatim, so a function blocked by two
different `fmt` calls counts toward neither: **these are a LOWER bound per
class**, deliberately, because collapsing callee strings into classes is
fragile and got it wrong once already.

**Population: hand-written code only.** Generated functions are excluded
here, because this table ranks work and generated code is not work anyone
does. Before that exclusion the 2026-07-21 measurement had this table's
`(*sync.Once).Do` row at 326 functions, 239 of them generated.

| Blocker | Functions |
|---|---|
| loop with unrecognized trip count | 625 |
| unresolved cost at call to fmt.Errorf | 341 |
| unresolved cost at call to fmt.Sprintf | 333 |
| unresolved cost at call to t6 | 196 |
| unresolved cost at call to (*sync.Once).Do | 90 |
| unresolved cost at call to (*sync.Pool).Get | 87 |
| unresolved cost at call to f | 79 |
| unresolved cost at call to (*net/rpc.Client).Call | 72 |
| unresolved cost at call to t1 | 70 |
| unresolved cost at call to encoding/json.Marshal | 69 |
| unresolved argument size at call to copy | 56 |
| unresolved cost at call to github.com/google/go-dap.WriteProtocolMessage | 48 |
| unresolved cost at call to google.golang.org/grpc.newFuncDialOption | 45 |
| goroutine launch (concurrency is unverifiable in v1) | 44 |
| unresolved cost at call to (github.com/jackc/pgx/v5/pgtype.EncodePlan).Encode | 44 |
| unresolved cost at call to encoding/json.Unmarshal | 43 |
| unresolved cost at call to t4 | 37 |
| unresolved cost at call to (context.Context).Value | 35 |
| unresolved cost at call to github.com/gohugoio/hugo/tpl/internal.AddTemplateFuncsNamespace | 30 |
| unresolved cost at call to google.golang.org/grpc.newFuncServerOption | 29 |
| unresolved cost at call to (*github.com/gohugoio/hugo/common/paths.Path).norm | 27 |
| unresolved cost at call to t2 | 27 |
| unresolved cost at call to (github.com/jackc/pgx/v5/pgtype.ScanPlan).Scan | 25 |
| unresolved cost at call to (*github.com/prometheus/common/config.HTTPClientConfig).SetDirectory | 24 |
| unresolved cost at call to (*time.Timer).Stop | 22 |
| unresolved cost at call to (error).Error | 22 |
| unresolved cost at call to (*strings.Builder).WriteString | 21 |
| unresolved cost at call to t10 | 20 |
| unresolved cost at call to bytes.Repeat | 19 |
| unresolved cost at call to time.AfterFunc | 18 |

## Top 30 blockers by SITES

**A concentration measure, not a work queue.** It shows where unverifiability
clusters, never whether that blocker can be removed — the two 2026-07-20
probes worked this ranking from the top down and produced no engine slice
(`fmt` 8,367 sites → 298 priceable functions; function values 2,878 → zero).
Rank work by the table above; use this one to understand shape.

Population: hand-written code only, as above.

| Blocker | Sites |
|---|---|
| loop with unrecognized trip count | 43226 |
| unresolved cost at call to fmt.Errorf | 5058 |
| unresolved cost at call to fmt.Sprintf | 2192 |
| unresolved cost at call to (error).Error | 479 |
| unresolved cost at call to fmt.Fprintf | 474 |
| unresolved cost at call to (*github.com/caddyserver/caddy/v2/caddyconfig/caddyfile.Dispenser).ArgErr | 429 |
| unresolved cost at call to (*strings.Builder).WriteString | 426 |
| unresolved cost at call to path/filepath.Join | 369 |
| unresolved cost at call to (*github.com/nats-io/nats-server/v2/server.Server).jsonResponse | 340 |
| unresolved cost at call to (*github.com/caddyserver/caddy/v2/caddyconfig/caddyfile.Dispenser).Errf | 336 |
| goroutine launch (concurrency is unverifiable in v1) | 335 |
| unresolved cost at call to (*github.com/caddyserver/caddy/v2/caddyconfig/caddyfile.Dispenser).NextArg | 329 |
| unresolved cost at call to (reflect.Value).Interface | 327 |
| unresolved argument size at call to strings.HasPrefix | 318 |
| unresolved argument size at call to strings.ToLower | 299 |
| unresolved cost at call to (*github.com/pb33f/ordered-map/v2.OrderedMap[string, *github.com/pb33f/libopenapi/datamodel/high/base.SchemaProxy]).Set | 298 |
| unresolved cost at call to (*github.com/nats-io/nats-server/v2/server.Server).sendAPIErrResponse | 271 |
| unresolved cost at call to encoding/json.Marshal | 271 |
| unresolved argument size at call to copy | 270 |
| unresolved cost at call to (io.Writer).Write | 249 |
| unresolved cost at call to errors.Is | 243 |
| unresolved cost at call to (*bytes.Buffer).WriteString | 235 |
| unresolved cost at call to (github.com/google/cel-go/common/ast.Expr).ID | 231 |
| unresolved cost at call to github.com/nats-io/nats-server/v2/server.parseOpts | 222 |
| unresolved cost at call to encoding/json.Unmarshal | 218 |
| unresolved cost at call to (*testing.common).Helper | 201 |
| unresolved argument size at call to strings.Contains | 198 |
| unresolved cost at call to (*google.golang.org/grpc/internal/grpclog.PrefixLogger).Infof | 187 |
| unresolved cost at call to go.uber.org/zap.String | 186 |
| unresolved argument size at call to strings.Join | 185 |

## Advisory smells by rule

The OUTWARD-facing table: every other table here ranks what bigo should
build, and this one ranks what somebody else's code could be told about.
Population: first-party, hand-written. Counts only — the finding list is in
`survey.json`, and the triage sample is in `CONTRIB-QUEUE.md`.

A count is not a defect. Five of the eight rules have prior art in linters
Go projects already run (SM3 ≈ `prealloc`, SM1/SM7 ≈ `gocritic`,
SM4 ≈ `perfsprint`-adjacent), so volume here says more about how common a
pattern is than about whether a maintainer would merge the fix.

| Rule | Findings |
|---|---|
| SM3 | 129 |
| SM6 | 116 |
| SM5 | 35 |
| SM4 | 18 |
| SM1 | 16 |
| SM2 | 6 |
