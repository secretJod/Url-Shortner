# ADR-0003: Decouple the analytics worker into an independently deployable process

## Status
🔶 **Partially implemented.** The first phased step decided below — add `recover()` to the worker loop — is done (`worker.go`'s `run()` now recovers from panics and restarts itself; see `08-reliability.md`). The second step — extracting `cmd/worker/main.go` as an independently deployable, horizontally-scalable process — has **not** been done; the worker still runs as a goroutine inside the API process with a hardcoded single consumer (`worker-1`).

## Problem
The analytics worker runs as a goroutine inside the same OS process as the API server (`main.go:45-46`, `worker.Start(ctx)` → `go w.run(ctx)`), with no panic recovery (`08-reliability.md`) and no independent scaling (`worker.New()` hardcodes a single `ConsumerName: "worker-1"`, `07-scalability.md`). A crash or backlog in analytics processing currently risks (a) taking down the whole API process on an unrecovered panic, and (b) being unable to scale click-ingestion throughput independently of API request throughput.

## Options considered

1. **Leave as-is, just add `recover()`.** Cheapest fix, addresses the crash-coupling risk (`08-reliability.md` fix #2) but does not address independent scaling — at high click volume, the only way to add worker capacity is to run more full API replicas (wasteful — you don't need more API capacity, just more stream-draining capacity) or continue running exactly one consumer regardless of API replica count.
2. **Extract the worker into its own Go binary/container, sharing the `internal/redis` and `internal/db`/`internal/store` packages** (already structured as reusable, decoupled packages — no rewrite needed, just a new `cmd/worker/main.go` entrypoint and Compose service), able to run N replicas in the same `analytics-workers` consumer group for horizontal scale.
3. **Replace the custom Redis Stream worker with a managed queue product.** Rejected outright — every candidate is either paid or adds an external dependency with no clear benefit over the already-implemented, already-correct Redis Streams consumer-group pattern; violates the project's standard against blind architecture change without evidence of need, since the current mechanism is not shown to be broken, only under-scaled operationally.

## Decision
Adopt **Option 2**, phased: first ship the `recover()` fix from Option 1 immediately (cheap, no architecture change, closes an active reliability gap), then extract `cmd/worker/main.go` as a follow-up once approved, reusing the existing `internal/worker`, `internal/redis`, `internal/db` packages verbatim. Both are compatible, sequential steps, not competing choices.

## Trade-offs
- A second deployable process/container means one more thing to build, deploy, and monitor — acceptable given the analytics path is already architected as loosely coupled (fire-and-forget from the API's perspective); this ADR just finishes that separation at the process level, matching what the code's own module boundaries already imply.
- Within the ₹0/$0 constraint, an extra container is free on self-hosted Docker Compose but may need explicit inclusion when picking a managed-hosting free tier in `06-deployment.md` (e.g. does the chosen free-tier host support running two separate services from one repo at $0) — flagged as a dependency between this ADR and the deployment provider decision.
