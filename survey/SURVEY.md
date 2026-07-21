# bigo real-world survey

GENERATED — do not edit; regenerate with `task survey`.

**This is a MANUAL measurement, not a golden test.** No test asserts its
contents and CI never runs it. Its targets are repositories that exist on one
machine at whatever commit they happen to sit, so these numbers are a record
of one run — compare across runs only via the per-target commit below.

Run 2026-07-21 with bigo 1.37.0.

**Aggregate: 31.6%** — 10572 of 33504 first-party functions bounded.

**Hand-written: 30.2%** — 8832 of 29214 functions bounded, with 4290 generated functions excluded.

Generated code is first-party by module path and is real code, but nobody
hand-tunes it and its unverifiability is usually the CORRECT answer — the
2026-07-21 `(*sync.Once).Do` probe measured 239 of that class's 326
sole-blocker functions as generated protobuf whose verdict is right.
**The aggregate above is kept unrebased** so it stays comparable with the
2026-07-20/21 probes, which pin their population to it.

**Hand-written near frontier: 8337 of 20382 (40.9%), ceiling 58.8%.**

**Near frontier: 9966 of 22932 unverifiable functions (43.5%) sit within 2 distinct blockers of a bound.** Clearing all of them would put coverage at **61.3%** — an UPPER BOUND, not a forecast: clearing a blocker for one function need not clear it for another. Two 2026-07-20 probes measured that gap directly (`fmt`: 744 sole-blocker functions, 298 actually priceable; function values: 573, zero reachable).

## Per target

| Target | Module | Commit | Functions | Bounded | Coverage | Generated | Hand | Hand cov | Near | Ceiling |
|---|---|---|---|---|---|---|---|---|---|---|
| grpc-go | google.golang.org/grpc | 2fd426d0 | 5467 | 1901 | 34.8% | 1902 | 3565 | 28.0% | 1991 | 71.2% |
| caddy | github.com/caddyserver/caddy/v2 | 0e8eb41b | 1963 | 454 | 23.1% | 0 | 1963 | 23.1% | 475 | 47.3% |
| prometheus | github.com/prometheus/prometheus | a0524eeca | 5859 | 1848 | 31.5% | 776 | 5083 | 30.4% | 1773 | 61.8% |
| etcd | go.etcd.io/etcd/v3 | 22b4192b9 | 98 | 9 | 9.2% | 0 | 98 | 9.2% | 40 | 50.0% |
| delve | github.com/go-delve/delve | 8fc4acbd | 2793 | 709 | 25.4% | 28 | 2765 | 25.4% | 716 | 51.0% |
| chi | github.com/go-chi/chi/v5 | 3b17157 | 180 | 60 | 33.3% | 0 | 180 | 33.3% | 60 | 66.7% |
| goldmark | github.com/yuin/goldmark | 50ba9fc | 795 | 442 | 55.6% | 0 | 795 | 55.6% | 130 | 71.9% |
| pgx | github.com/jackc/pgx/v5 | 0a977a6 | 2099 | 750 | 35.7% | 110 | 1989 | 35.9% | 662 | 67.3% |
| cel-go | github.com/google/cel-go | 646511d | 3586 | 1503 | 41.9% | 937 | 2649 | 39.0% | 1122 | 73.2% |
| expr | github.com/expr-lang/expr | 4b31df3 | 1286 | 224 | 17.4% | 515 | 771 | 29.1% | 215 | 34.1% |
| nats-server | github.com/nats-io/nats-server/v2 | 2e5f51f31 | 4000 | 921 | 23.0% | 0 | 4000 | 23.0% | 1105 | 50.6% |
| hugo | github.com/gohugoio/hugo | 89b8c3220 | 5378 | 1751 | 32.6% | 22 | 5356 | 32.3% | 1677 | 63.7% |

## Distance to bound

How many DISTINCT leaf blockers stand between an unverifiable function and a
bound, walking through propagation. This is why a single headline coverage
number is misleading: it averages a near frontier that incremental work can
reach against a deep tail that no achievable engine work will.

| Blockers | Functions | Share |
|---|---|---|
| 0 | 22 | 0.1% |
| 1 | 6665 | 29.1% |
| 2 | 3279 | 14.3% |
| 3 | 2058 | 9.0% |
| 4 | 1662 | 7.2% |
| 5 | 1131 | 4.9% |
| 6 | 749 | 3.3% |
| 7 | 659 | 2.9% |
| 8 | 494 | 2.2% |
| 9 | 417 | 1.8% |
| 10+ | 5796 | 25.3% |

## Unverifiable by cause kind

| Cause | Count |
|---|---|
| call | 95650 |
| loop | 43226 |
| defer | 884 |
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
| loop with unrecognized trip count | 545 |
| unresolved cost at call to fmt.Sprintf | 322 |
| unresolved cost at call to fmt.Errorf | 294 |
| unresolved cost at call to t6 | 191 |
| unresolved cost at call to (*sync.Once).Do | 85 |
| unresolved cost at call to f | 75 |
| unresolved cost at call to (*net/rpc.Client).Call | 72 |
| unresolved cost at call to encoding/json.Marshal | 69 |
| unresolved cost at call to t1 | 65 |
| unresolved cost at call to github.com/google/go-dap.WriteProtocolMessage | 48 |
| unresolved argument size at call to copy | 46 |
| unresolved cost at call to google.golang.org/grpc.newFuncDialOption | 45 |
| unresolved cost at call to (github.com/jackc/pgx/v5/pgtype.EncodePlan).Encode | 44 |
| goroutine launch (concurrency is unverifiable in v1) | 43 |
| unresolved cost at call to encoding/json.Unmarshal | 43 |
| unresolved cost at call to t4 | 37 |
| unresolved cost at call to (*sync.Pool).Get | 36 |
| unresolved cost at call to (context.Context).Value | 34 |
| unresolved cost at call to (*sync/atomic.Bool).Load | 33 |
| unresolved cost at call to github.com/gohugoio/hugo/tpl/internal.AddTemplateFuncsNamespace | 30 |
| unresolved cost at call to context.WithValue | 29 |
| unresolved cost at call to google.golang.org/grpc.newFuncServerOption | 29 |
| unresolved cost at call to (*github.com/gohugoio/hugo/common/paths.Path).norm | 27 |
| unresolved cost at call to (*sync.Pool).Put | 27 |
| unresolved cost at call to t2 | 26 |
| unresolved cost at call to (github.com/jackc/pgx/v5/pgtype.ScanPlan).Scan | 25 |
| unresolved cost at call to (*github.com/prometheus/common/config.HTTPClientConfig).SetDirectory | 24 |
| unresolved cost at call to (error).Error | 22 |
| unresolved cost at call to (*strings.Builder).WriteString | 21 |
| unresolved cost at call to (encoding/binary.bigEndian).Uint16 | 20 |

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
| unresolved argument size at call to strings.Contains | 191 |
| unresolved cost at call to (*google.golang.org/grpc/internal/grpclog.PrefixLogger).Infof | 187 |
| unresolved cost at call to go.uber.org/zap.String | 186 |
| unresolved cost at call to (*log/slog.Logger).Error | 185 |
