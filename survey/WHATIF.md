# bigo what-if — 2026-07-24 (bigo 1.39.1)

Baseline: 33504 first-party functions, 10571 bounded.

Graduations are EXACT engine results under each candidate's assumption
set — tainted ⊤→bounded transitions only; assumed targets excluded.
A graduation says the propagation clears, NOT that the assumed bound is
truthful — that half needs a probe or an implementation argument (spec §6).

| candidate | graduated | hand-written | Δ coverage |
|---|---|---|---|
| S1-truthful | 61 | 61 | +0.18pp |
| S1b-atomic-family | 163 | 163 | +0.49pp |

## S1-truthful

- grpc-go: 6 graduated (6 hand-written)
- caddy: 1 graduated (1 hand-written)
- prometheus: 2 graduated (2 hand-written)
- etcd: 0 graduated (0 hand-written)
- delve: 2 graduated (2 hand-written)
- chi: 0 graduated (0 hand-written)
- goldmark: 0 graduated (0 hand-written)
- pgx: 0 graduated (0 hand-written)
- cel-go: 4 graduated (4 hand-written)
- expr: 0 graduated (0 hand-written)
- nats-server: 38 graduated (38 hand-written)
- hugo: 8 graduated (8 hand-written)
- warning: (*sync.Pool).Put (absent from 4 of 12 targets)

## S1b-atomic-family

- grpc-go: 39 graduated (39 hand-written)
- caddy: 13 graduated (13 hand-written)
- prometheus: 4 graduated (4 hand-written)
- etcd: 0 graduated (0 hand-written)
- delve: 5 graduated (5 hand-written)
- chi: 1 graduated (1 hand-written)
- goldmark: 0 graduated (0 hand-written)
- pgx: 0 graduated (0 hand-written)
- cel-go: 9 graduated (9 hand-written)
- expr: 0 graduated (0 hand-written)
- nats-server: 72 graduated (72 hand-written)
- hugo: 20 graduated (20 hand-written)
- warning: (*sync/atomic.Int32).Load (absent from 2 of 12 targets)
- warning: (*sync/atomic.Int32).Store (absent from 2 of 12 targets)
- warning: (*sync/atomic.Int32).Swap (absent from 2 of 12 targets)
- warning: (*sync/atomic.Int32).CompareAndSwap (absent from 2 of 12 targets)
- warning: (*sync/atomic.Int32).Add (absent from 2 of 12 targets)
- warning: (*sync/atomic.Int32).And (absent from 2 of 12 targets)
- warning: (*sync/atomic.Int32).Or (absent from 2 of 12 targets)
- warning: (*sync/atomic.Int64).Load (absent from 4 of 12 targets)
- warning: (*sync/atomic.Int64).Store (absent from 4 of 12 targets)
- warning: (*sync/atomic.Int64).Swap (absent from 4 of 12 targets)
- warning: (*sync/atomic.Int64).CompareAndSwap (absent from 4 of 12 targets)
- warning: (*sync/atomic.Int64).Add (absent from 4 of 12 targets)
- warning: (*sync/atomic.Int64).And (absent from 4 of 12 targets)
- warning: (*sync/atomic.Int64).Or (absent from 4 of 12 targets)
- warning: (*sync/atomic.Uint32).Load (absent from 6 of 12 targets)
- warning: (*sync/atomic.Uint32).Store (absent from 6 of 12 targets)
- warning: (*sync/atomic.Uint32).Swap (absent from 6 of 12 targets)
- warning: (*sync/atomic.Uint32).CompareAndSwap (absent from 6 of 12 targets)
- warning: (*sync/atomic.Uint32).Add (absent from 6 of 12 targets)
- warning: (*sync/atomic.Uint32).And (absent from 6 of 12 targets)
- warning: (*sync/atomic.Uint32).Or (absent from 6 of 12 targets)
- warning: (*sync/atomic.Uint64).Load (absent from 2 of 12 targets)
- warning: (*sync/atomic.Uint64).Store (absent from 2 of 12 targets)
- warning: (*sync/atomic.Uint64).Swap (absent from 2 of 12 targets)
- warning: (*sync/atomic.Uint64).CompareAndSwap (absent from 2 of 12 targets)
- warning: (*sync/atomic.Uint64).Add (absent from 2 of 12 targets)
- warning: (*sync/atomic.Uint64).And (absent from 2 of 12 targets)
- warning: (*sync/atomic.Uint64).Or (absent from 2 of 12 targets)
- warning: (*sync.Pool).Put (absent from 4 of 12 targets)
