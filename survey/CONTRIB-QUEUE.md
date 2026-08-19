# bigo contribution lane — triage queue

GENERATED — do not edit; regenerate with `task contrib-scan`.

**A MANUAL measurement, not a golden test.** No test asserts its contents and
CI never runs it. Targets are repositories on one machine at whatever commit
they happen to sit; compare runs only via the per-target commit below.

Run 2026-08-19 with bigo 1.46.0.

**Sample: 40 findings, at most 8 per rule and 6 per target**, drawn in target order then file, line, rule. The rule was registered in `docs/bigo/investigations/2026-08-18-contribution-lane-thresholds.md` before this scan ran, and is implemented in `survey.Sample` so the two cannot drift. The per-target cap is Amendment 1, made before any verdict was assigned.

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
| 1 | caddy | SM6 | `admin.go:301` | map built without a size hint in a loop bounded by O(len(admin.Origins)); preallocate with make(map[K]V, O(len(admin.Origins))) | not-sendable |
| 2 | caddy | SM1 | `caddyconfig/caddyfile/lexer.go:317` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | sendable |
| 3 | caddy | SM3 | `caddyconfig/caddyfile/parse.go:798` | append in a loop bounded by O(len(sb.Segments)) on a zero-capacity slice; preallocate with make(…, 0, O(len(sb.Segments))) | FP |
| 4 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:83` | map built without a size hint in a loop bounded by O(len(originalServerBlocks)); preallocate with make(map[K]V, O(len(originalServerBlocks))) | not-sendable |
| 5 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:96` | map built without a size hint in a loop bounded by O(len(originalServerBlocks)); preallocate with make(map[K]V, O(len(originalServerBlocks))) | FP |
| 6 | caddy | SM6 | `caddyconfig/httpcaddyfile/addresses.go:118` | map built without a size hint in a loop bounded by O(len(originalServerBlocks)); preallocate with make(map[K]V, O(len(originalServerBlocks))) | FP |
| 7 | nats-server | SM6 | `server/accounts.go:708` | map built without a size hint in a loop bounded by O(len(dests)); preallocate with make(map[K]V, O(len(dests))) | not-sendable |
| 8 | nats-server | SM3 | `server/certidp/certidp.go:196` | append in a loop bounded by O(len(uris)) on a zero-capacity slice; preallocate with make(…, 0, O(len(uris))) | not-sendable |
| 9 | nats-server | SM3 | `server/certstore/certstore_windows.go:179` | append in a loop bounded by O(len(caCertsMatch)) on a zero-capacity slice; preallocate with make(…, 0, O(len(caCertsMatch))) | not-sendable |
| 10 | nats-server | SM3 | `server/certstore/certstore_windows.go:432` | append in a loop bounded by O(1) on a zero-capacity slice; preallocate with make(…, 0, O(1)) | FP |
| 11 | nats-server | SM3 | `server/client.go:6713` | append in a loop bounded by O(len(cts)) on a zero-capacity slice; preallocate with make(…, 0, O(len(cts))) | not-sendable |
| 12 | nats-server | SM1 | `server/errors.go:319` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | not-sendable |
| 13 | hugo | SM3 | `codegen/methods.go:357` | append in a loop bounded by O(len(m)) on a zero-capacity slice; preallocate with make(…, 0, O(len(m))) | not-sendable |
| 14 | hugo | SM4 | `codegen/methods.go:381` | regexp compiled inside a loop; hoist the pattern | FP |
| 15 | hugo | SM3 | `codegen/methods.go:402` | append in a loop bounded by O(len(m)) on a zero-capacity slice; preallocate with make(…, 0, O(len(m))) | not-sendable |
| 16 | hugo | SM3 | `codegen/methods.go:449` | append in a loop bounded by O(len(f)) on a zero-capacity slice; preallocate with make(…, 0, O(len(f))) | not-sendable |
| 17 | hugo | SM6 | `codegen/methods.go:510` | map built without a size hint in a loop bounded by O(len(s)); preallocate with make(map[K]V, O(len(s))) | not-sendable |
| 18 | hugo | SM6 | `commands/convert.go:206` | map built without a size hint in a loop bounded by O(len(pagesBackedByFile)); preallocate with make(map[K]V, O(len(pagesBackedByFile))) | FP |
| 19 | prometheus | SM6 | `cmd/promtool/analyze.go:95` | map built without a size hint in a loop bounded by O(len(matchers)); preallocate with make(map[K]V, O(len(matchers))) | FP |
| 20 | prometheus | SM5 | `cmd/promtool/unittest.go:412` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
| 21 | prometheus | SM5 | `cmd/promtool/unittest.go:413` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
| 22 | prometheus | SM5 | `cmd/promtool/unittest.go:517` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
| 23 | prometheus | SM5 | `cmd/promtool/unittest.go:520` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
| 24 | prometheus | SM5 | `discovery/moby/docker.go:258` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
| 25 | grpc-go | SM1 | `internal/resolver/dns/dns_resolver.go:303` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | not-sendable |
| 26 | grpc-go | SM5 | `profiling/cmd/catapult.go:324` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
| 27 | grpc-go | SM1 | `xds/internal/xdsclient/xdsresource/matcher.go:129` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | not-sendable |
| 28 | grpc-go | SM4 | `xds/internal/xdsclient/xdsresource/unmarshal_rds.go:237` | regexp compiled inside a loop; hoist the pattern | FP |
| 29 | grpc-go | SM4 | `xds/internal/xdsclient/xdsresource/unmarshal_rds.go:257` | regexp compiled inside a loop; hoist the pattern | FP |
| 30 | grpc-go | SM4 | `xds/internal/xdsclient/xdsresource/unmarshal_rds.go:415` | regexp compiled inside a loop; hoist the pattern | FP |
| 31 | goldmark | SM2 | `testutil/testutil.go:167` | repeated linear scan over the same slice (quadratic); build a map/set once before the loop | not-sendable |
| 32 | cel-go | SM5 | `cel/prompt.go:159` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
| 33 | cel-go | SM1 | `common/error.go:61` | string built by repeated concatenation in a loop (quadratic); use strings.Builder | not-sendable |
| 34 | delve | SM5 | `pkg/proc/bininfo.go:268` | sort inside a data-dependent loop (composed O(n·m log m)); hoist or restructure | FP |
