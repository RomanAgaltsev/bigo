# bigo real-world survey

GENERATED — do not edit; regenerate with `task survey`.

**This is a MANUAL measurement, not a golden test.** No test asserts its
contents and CI never runs it. Its targets are repositories that exist on one
machine at whatever commit they happen to sit, so these numbers are a record
of one run — compare across runs only via the per-target commit below.

Run 2026-07-20 with bigo 1.36.0.

**Aggregate: 31.6%** — 10572 of 33504 first-party functions bounded.

**Near frontier: 9966 of 22932 unverifiable functions (43.5%) sit within 2 distinct blockers of a bound.** Clearing all of them would put coverage at **61.3%** — an UPPER BOUND, not a forecast: clearing a blocker for one function need not clear it for another. Two 2026-07-20 probes measured that gap directly (`fmt`: 744 sole-blocker functions, 298 actually priceable; function values: 573, zero reachable).

## Per target

| Target | Module | Commit | Functions | Bounded | Coverage | Near | Ceiling |
|---|---|---|---|---|---|---|---|
| grpc-go | google.golang.org/grpc | 2fd426d0 | 5467 | 1901 | 34.8% | 1991 | 71.2% |
| caddy | github.com/caddyserver/caddy/v2 | 0e8eb41b | 1963 | 454 | 23.1% | 475 | 47.3% |
| prometheus | github.com/prometheus/prometheus | a0524eeca | 5859 | 1848 | 31.5% | 1773 | 61.8% |
| etcd | go.etcd.io/etcd/v3 | 22b4192b9 | 98 | 9 | 9.2% | 40 | 50.0% |
| delve | github.com/go-delve/delve | 8fc4acbd | 2793 | 709 | 25.4% | 716 | 51.0% |
| chi | github.com/go-chi/chi/v5 | 3b17157 | 180 | 60 | 33.3% | 60 | 66.7% |
| goldmark | github.com/yuin/goldmark | 50ba9fc | 795 | 442 | 55.6% | 130 | 71.9% |
| pgx | github.com/jackc/pgx/v5 | 0a977a6 | 2099 | 750 | 35.7% | 662 | 67.3% |
| cel-go | github.com/google/cel-go | 646511d | 3586 | 1503 | 41.9% | 1122 | 73.2% |
| expr | github.com/expr-lang/expr | 4b31df3 | 1286 | 224 | 17.4% | 215 | 34.1% |
| nats-server | github.com/nats-io/nats-server/v2 | 2e5f51f31 | 4000 | 921 | 23.0% | 1105 | 50.6% |
| hugo | github.com/gohugoio/hugo | 89b8c3220 | 5378 | 1751 | 32.6% | 1677 | 63.7% |

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
| call | 102137 |
| loop | 47965 |
| defer | 884 |
| go | 335 |
| nobody | 3 |

Compare with `corpus/CORPUS.md`: the canonical corpus and real code do not
agree on this ranking, and real code is the one that reflects adoption.

## Top 30 blockers by GRADUATION count

**This table is the deliverable.** It counts functions whose ONLY blocker is
each entry — the number that would actually graduate if it were cleared.

A class here is one cause detail verbatim, so a function blocked by two
different `fmt` calls counts toward neither: **these are a LOWER bound per
class**, deliberately, because collapsing callee strings into classes is
fragile and got it wrong once already.

| Blocker | Functions |
|---|---|
| loop with unrecognized trip count | 546 |
| unresolved cost at call to fmt.Sprintf | 377 |
| unresolved cost at call to (*sync.Once).Do | 326 |
| unresolved cost at call to fmt.Errorf | 300 |
| unresolved cost at call to t6 | 191 |
| unresolved cost at call to (google.golang.org/protobuf/internal/impl.Export).MessageStringOf | 187 |
| unresolved cost at call to math/bits.Len64 | 79 |
| unresolved cost at call to f | 75 |
| unresolved cost at call to (*net/rpc.Client).Call | 72 |
| unresolved cost at call to encoding/json.Marshal | 69 |
| unresolved cost at call to (*github.com/antlr4-go/antlr/v4.BaseParserRuleContext).GetToken | 67 |
| unresolved cost at call to t1 | 65 |
| unresolved cost at call to github.com/antlr4-go/antlr/v4.InitBaseParserRuleContext | 51 |
| unresolved cost at call to github.com/google/go-dap.WriteProtocolMessage | 48 |
| unresolved argument size at call to copy | 46 |
| unresolved cost at call to encoding/json.Unmarshal | 46 |
| unresolved cost at call to google.golang.org/grpc.newFuncDialOption | 45 |
| unresolved cost at call to (github.com/jackc/pgx/v5/pgtype.EncodePlan).Encode | 44 |
| goroutine launch (concurrency is unverifiable in v1) | 43 |
| unresolved cost at call to (*github.com/gogo/protobuf/proto.InternalMessageInfo).DiscardUnknown | 37 |
| unresolved cost at call to (*github.com/gogo/protobuf/proto.InternalMessageInfo).Merge | 37 |
| unresolved cost at call to github.com/gogo/protobuf/proto.CompactTextString | 37 |
| unresolved cost at call to t4 | 37 |
| unresolved cost at call to (*github.com/antlr4-go/antlr/v4.BaseParseTreeVisitor).VisitChildren | 36 |
| unresolved cost at call to (*sync.Pool).Get | 36 |
| unresolved cost at call to (context.Context).Value | 34 |
| unresolved cost at call to (*sync/atomic.Bool).Load | 33 |
| unresolved cost at call to (google.golang.org/grpc.ClientConnInterface).Invoke | 33 |
| unresolved cost at call to github.com/gohugoio/hugo/tpl/internal.AddTemplateFuncsNamespace | 30 |
| unresolved cost at call to context.WithValue | 29 |

## Top 30 blockers by SITES

**A concentration measure, not a work queue.** It shows where unverifiability
clusters, never whether that blocker can be removed — the two 2026-07-20
probes worked this ranking from the top down and produced no engine slice
(`fmt` 8,367 sites → 298 priceable functions; function values 2,878 → zero).
Rank work by the table above; use this one to understand shape.

| Blocker | Sites |
|---|---|
| loop with unrecognized trip count | 47965 |
| unresolved cost at call to fmt.Errorf | 5431 |
| unresolved cost at call to fmt.Sprintf | 2204 |
| unresolved cost at call to (interface).Helper | 573 |
| unresolved cost at call to (error).Error | 479 |
| unresolved cost at call to fmt.Fprintf | 474 |
| unresolved cost at call to (*github.com/caddyserver/caddy/v2/caddyconfig/caddyfile.Dispenser).ArgErr | 429 |
| unresolved cost at call to (*strings.Builder).WriteString | 426 |
| unresolved cost at call to (*google.golang.org/protobuf/internal/impl.messageState).StoreMessageInfo | 374 |
| unresolved cost at call to (google.golang.org/protobuf/internal/impl.Export).MessageStateOf | 374 |
| unresolved cost at call to path/filepath.Join | 369 |
| unresolved cost at call to (*github.com/nats-io/nats-server/v2/server.Server).jsonResponse | 340 |
| unresolved cost at call to (*github.com/caddyserver/caddy/v2/caddyconfig/caddyfile.Dispenser).Errf | 336 |
| goroutine launch (concurrency is unverifiable in v1) | 335 |
| unresolved cost at call to (*github.com/caddyserver/caddy/v2/caddyconfig/caddyfile.Dispenser).NextArg | 329 |
| unresolved cost at call to (reflect.Value).Interface | 327 |
| unresolved argument size at call to strings.HasPrefix | 318 |
| unresolved argument size at call to strings.ToLower | 299 |
| unresolved cost at call to (*github.com/pb33f/ordered-map/v2.OrderedMap[string, *github.com/pb33f/libopenapi/datamodel/high/base.SchemaProxy]).Set | 298 |
| unresolved argument size at call to copy | 272 |
| unresolved cost at call to (*github.com/nats-io/nats-server/v2/server.Server).sendAPIErrResponse | 271 |
| unresolved cost at call to encoding/json.Marshal | 271 |
| unresolved cost at call to (io.Writer).Write | 249 |
| unresolved cost at call to errors.Is | 243 |
| unresolved cost at call to (*bytes.Buffer).WriteString | 235 |
| unresolved cost at call to (github.com/google/cel-go/common/ast.Expr).ID | 231 |
| unresolved cost at call to github.com/nats-io/nats-server/v2/server.parseOpts | 222 |
| unresolved cost at call to encoding/json.Unmarshal | 221 |
| unresolved cost at call to (*testing.common).Helper | 201 |
| unresolved argument size at call to strings.Contains | 191 |
