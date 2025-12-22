# Roadmap

`recourse` is being built in phases to keep the public surface small while the internals evolve.

## Current status

Implemented today:

- Basic retry executor (bounded attempts, backoff/jitter, per-attempt + overall timeouts)
- Policy schema + normalization/clamping
- Static in-process policy provider
- Timelines (`observe.Timeline`) and observer callbacks (`observe.Observer`)
- Outcome classifiers (`classify.Classifier`) selected by policy
- Budgets/backpressure (per-attempt gating via `budget` + `ExecutorOptions.Budgets`, fail-closed defaults)
- Facade helpers that accept string keys (`recourse.Do*`)
- API Ergonomics: Functional options for policies/executors and timeline capture context
- Hedging: Fixed-delay and Latency-Aware (P99) triggers
- Circuit Breaking: Consecutive failure breaker with half-open recovery

## Planned phases (high level)

### Remote control plane

Add a remote policy provider with caching, last-known-good fallback, and negative caching for missing keys.

### Integrations and ergonomics

Add common integrations (HTTP/gRPC helpers) and stable zero-config wiring while preserving an explicit-executor path for advanced users.

### Hardening and v1 docs

Add race/leak hardening, broader tests, examples, and documentation polish leading to a v1.0 API freeze.
