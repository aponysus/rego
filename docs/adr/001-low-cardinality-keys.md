# ADR-001: Low-Cardinality Policy Keys

## Status
Accepted

## Context
recourse uses a policy key to select retry behavior, budgets, and observability metadata.
Keys are also used as dimensions for metrics, caches, and in-memory trackers.
If keys are high-cardinality (for example, include user IDs or request IDs),
they can cause unbounded memory growth, noisy metrics, and poor system behavior.

## Decision
Policy keys must be low-cardinality and stable over time.
Keys should represent a logical call site or operation (for example, "payments.Charge"),
not per-request or per-entity identifiers.

recourse treats the key as a primary dimension for policy selection and observability,
so the design assumes it is safe to aggregate on that dimension.

## Consequences
- Call sites must map dynamic values to stable categories before constructing a key.
- Metrics and caches remain bounded and interpretable.
- If a caller violates this rule, it can lead to elevated memory use and unusable metrics.
- Documentation and examples should reinforce best practices for key selection.
