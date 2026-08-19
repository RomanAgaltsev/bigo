# bigo contribution lane — triage queue

GENERATED — do not edit; regenerate with `task contrib-scan`.

**A MANUAL measurement, not a golden test.** No test asserts its contents and
CI never runs it. Targets are repositories on one machine at whatever commit
they happen to sit; compare runs only via the per-target commit below.

Run 2026-08-19 with bigo 1.46.0.

**Sample: 40 findings, at most 8 per rule**, drawn in target order then file, line, rule. The rule was registered in `docs/bigo/investigations/2026-08-18-contribution-lane-thresholds.md` before this scan ran, and is implemented in `survey.Sample` so the two cannot drift.

Population: first-party, hand-written. Generated code is excluded — nobody
hand-tunes it, so a finding there is not a contribution.

## Targets scanned

| Target | Module | Commit | Findings |
|---|---|---|---|
| caddy | github.com/caddyserver/caddy/v2 | 0e8eb41b | 42 |
| nats-server | github.com/nats-io/nats-server/v2 | 2e5f51f31 | 28 |
| hugo | github.com/gohugoio/hugo | 89b8c3220 | 73 |
| prometheus | github.com/prometheus/prometheus | a0524eeca | 76 |
| grpc-go | google.golang.org/grpc | 2fd426d0 | 50 |
| pgx | github.com/jackc/pgx/v5 | 0a977a6 | 4 |
| goldmark | github.com/yuin/goldmark | 50ba9fc | 4 |
| cel-go | github.com/google/cel-go | 646511d | 7 |
| delve | github.com/go-delve/delve | 8fc4acbd | 18 |
| chi | github.com/go-chi/chi/v5 | 3b17157 | 2 |

## Findings by rule (whole population, not the sample)

| Rule | Findings |
|---|---|
| SM3 | 122 |
| SM6 | 115 |
| SM5 | 34 |
| SM4 | 18 |
| SM1 | 9 |
| SM2 | 6 |

## The sample

Verdict column is filled by hand during triage: `sendable`, `not-sendable`,
or `FP`. Read the target's own `.golangci.yml` before judging anything
sendable — a finding its CI already declines to enforce is not-sendable
however correct it is.

| # | Target | Rule | Position | Message | Verdict |
|---|---|---|---|---|---|
| 1 | caddy | SM6 | `admin.go:301` | map built without a size hint in a loop bounded by O(len(admin.Origins)); preallocate with make(map[K]V, O(len(admin.Origins))) | |
| 2 | caddy | SM1 | `caddyconfig/caddyfile/lexer.go:317` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 3 | caddy | SM3 | `caddyconfig/caddyfile/parse.go:798` | append in a loop bounded by O(len(sb.Segments)) on a zero-capacity slice; preallocate with make(…, 0, O(len(sb.Segments))) | |
| 4 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:83` | map built without a size hint in a loop bounded by O(len(originalServerBlocks)); preallocate with make(map[K]V, O(len(originalServerBlocks))) | |
| 5 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:96` | map built without a size hint in a loop bounded by O(len(originalServerBlocks)); preallocate with make(map[K]V, O(len(originalServerBlocks))) | |
| 6 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:118` | map built without a size hint in a loop bounded by O(len(originalServerBlocks)); preallocate with make(map[K]V, O(len(originalServerBlocks))) | |
| 7 | caddy | SM5 | `caddyconfig/httpcaddyfile/addresses.go:139` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 8 | caddy | SM5 | `caddyconfig/httpcaddyfile/addresses.go:153` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 9 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:157` | map built without a size hint in a loop bounded by O(len(originalServerBlocks)); preallocate with make(map[K]V, O(len(originalServerBlocks))) | |
| 10 | caddy | SM5 | `caddyconfig/httpcaddyfile/addresses.go:211` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 11 | caddy | SM5 | `caddyconfig/httpcaddyfile/addresses.go:240` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 12 | caddy | SM5 | `caddyconfig/httpcaddyfile/addresses.go:250` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 13 | caddy | SM6 | `caddyconfig/httpcaddyfile/directives.go:226` | map built without a size hint in a loop bounded by O(len(h.parentBlock.Segments)); preallocate with make(map[K]V, O(len(h.parentBlock.Segments))) | |
| 14 | caddy | SM6 | `caddyconfig/httpcaddyfile/directives.go:564` | map built without a size hint in a loop bounded by O(len(sb.parsedKeys)); preallocate with make(map[K]V, O(len(sb.parsedKeys))) | |
| 15 | caddy | SM6 | `caddyconfig/httpcaddyfile/directives.go:596` | map built without a size hint in a loop bounded by O(len(sb.parsedKeys)); preallocate with make(map[K]V, O(len(sb.parsedKeys))) | |
| 16 | caddy | SM5 | `caddyconfig/httpcaddyfile/httptype.go:353` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 17 | caddy | SM5 | `caddyconfig/httpcaddyfile/httptype.go:657` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 18 | caddy | SM5 | `caddyconfig/httpcaddyfile/httptype.go:781` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | |
| 19 | caddy | SM4 | `modules/caddyhttp/headers/headers.go:152` | regexp compiled inside a loop; hoist the pattern | |
| 20 | caddy | SM4 | `modules/caddyhttp/headers/headers.go:302` | regexp compiled inside a loop; hoist the pattern | |
| 21 | caddy | SM4 | `modules/caddyhttp/headers/headers.go:329` | regexp compiled inside a loop; hoist the pattern | |
| 22 | caddy | SM4 | `modules/caddyhttp/map/map.go:78` | regexp compiled inside a loop; hoist the pattern | |
| 23 | caddy | SM3 | `modules/caddyhttp/reverseproxy/selectionpolicies.go:143` | append in a loop bounded by O(len(r.Weights)) on a zero-capacity slice; preallocate with make(…, 0, O(len(r.Weights))) | |
| 24 | caddy | SM3 | `modules/caddyhttp/reverseproxy/selectionpolicies.go:823` | append in a loop bounded by O(len(upstreams)) on a zero-capacity slice; preallocate with make(…, 0, O(len(upstreams))) | |
| 25 | caddy | SM3 | `modules/caddyhttp/reverseproxy/upstreams.go:516` | append in a loop bounded by O(len(mu.sources)) on a zero-capacity slice; preallocate with make(…, 0, O(len(mu.sources))) | |
| 26 | caddy | SM4 | `modules/caddyhttp/rewrite/rewrite.go:114` | regexp compiled inside a loop; hoist the pattern | |
| 27 | caddy | SM3 | `modules/caddyhttp/server.go:1096` | append in a loop bounded by O(len(headers)) on a zero-capacity slice; preallocate with make(…, 0, O(len(headers))) | |
| 28 | caddy | SM4 | `modules/logging/filters.go:750` | regexp compiled inside a loop; hoist the pattern | |
| 29 | nats-server | SM3 | `server/certidp/certidp.go:196` | append in a loop bounded by O(len(uris)) on a zero-capacity slice; preallocate with make(…, 0, O(len(uris))) | |
| 30 | nats-server | SM3 | `server/certstore/certstore_windows.go:179` | append in a loop bounded by O(len(caCertsMatch)) on a zero-capacity slice; preallocate with make(…, 0, O(len(caCertsMatch))) | |
| 31 | nats-server | SM3 | `server/certstore/certstore_windows.go:432` | append in a loop bounded by O(1) on a zero-capacity slice; preallocate with make(…, 0, O(1)) | |
| 32 | nats-server | SM1 | `server/errors.go:319` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 33 | nats-server | SM1 | `server/errors.go:322` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 34 | nats-server | SM4 | `server/opts.go:5188` | regexp compiled inside a loop; hoist the pattern | |
| 35 | nats-server | SM2 | `server/reload.go:2811` | repeated linear scan over the same slice (quadratic); build a map/set once before the loop | |
| 36 | nats-server | SM2 | `server/reload.go:2818` | repeated linear scan over the same slice (quadratic); build a map/set once before the loop | |
| 37 | hugo | SM4 | `codegen/methods.go:381` | regexp compiled inside a loop; hoist the pattern | |
| 38 | hugo | SM2 | `langs/config.go:148` | repeated linear scan over the same slice (quadratic); build a map/set once before the loop | |
| 39 | hugo | SM1 | `resources/internal/resourcepaths.go:53` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
| 40 | hugo | SM1 | `resources/transform.go:451` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | |
