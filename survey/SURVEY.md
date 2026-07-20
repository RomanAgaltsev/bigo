# bigo real-world survey

GENERATED — do not edit; regenerate with `task survey`.

**This is a MANUAL measurement, not a golden test.** No test asserts its
contents and CI never runs it. Its targets are repositories that exist on one
machine at whatever commit they happen to sit, so these numbers are a record
of one run — compare across runs only via the per-target commit below.

Run 2026-07-20 with bigo 1.35.0.

**Aggregate: 31.6%** — 10572 of 33504 first-party functions bounded.

## Per target

| Target | Module | Commit | Functions | Bounded | Coverage |
|---|---|---|---|---|---|
| grpc-go | google.golang.org/grpc | 2fd426d0 | 5467 | 1901 | 34.8% |
| caddy | github.com/caddyserver/caddy/v2 | 0e8eb41b | 1963 | 454 | 23.1% |
| prometheus | github.com/prometheus/prometheus | a0524eeca | 5859 | 1848 | 31.5% |
| etcd | go.etcd.io/etcd/v3 | 22b4192b9 | 98 | 9 | 9.2% |
| delve | github.com/go-delve/delve | 8fc4acbd | 2793 | 709 | 25.4% |
| chi | github.com/go-chi/chi/v5 | 3b17157 | 180 | 60 | 33.3% |
| goldmark | github.com/yuin/goldmark | 50ba9fc | 795 | 442 | 55.6% |
| pgx | github.com/jackc/pgx/v5 | 0a977a6 | 2099 | 750 | 35.7% |
| cel-go | github.com/google/cel-go | 646511d | 3586 | 1503 | 41.9% |
| expr | github.com/expr-lang/expr | 4b31df3 | 1286 | 224 | 17.4% |
| nats-server | github.com/nats-io/nats-server/v2 | 2e5f51f31 | 4000 | 921 | 23.0% |
| hugo | github.com/gohugoio/hugo | 89b8c3220 | 5378 | 1751 | 32.6% |

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

## Top 30 blockers by detail

**This table is the deliverable.** It ranks work by what actually stops bigo
on real code, rather than by what the self-authored corpus happens to contain.

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
