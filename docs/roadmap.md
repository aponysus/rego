# Roadmap

`rego` is being built in phases to keep the public surface small while the internals evolve.

## Current status

Implemented today:

- Basic retry executor (bounded attempts, backoff/jitter, per-attempt + overall timeouts)
- Policy schema + normalization/clamping
- Static in-process policy provider
- Timelines (`observe.Timeline`) and observer callbacks (`observe.Observer`)
- Facade helpers that accept string keys (`rego.Do*`)

## Planned phases (high level)

### Outcomes and classifiers

Add protocol/domain-aware retry decisions (beyond “retry on error”), driven by a classifier registry and referenced by policy.

### Budgets

Add per-attempt budgeting/backpressure to prevent retry/hedge storms.

### Hedging

Add fixed-delay hedging, then latency-aware triggers, with strict concurrency/cancellation guarantees.

### Remote control plane

Add a remote policy provider with caching, last-known-good fallback, and negative caching for missing keys.

### Integrations and ergonomics

Add common integrations (HTTP/gRPC helpers) and stable zero-config wiring while preserving an explicit-executor path for advanced users.

### Hardening and v1 docs

Add race/leak hardening, broader tests, examples, and documentation polish leading to a v1.0 API freeze.

