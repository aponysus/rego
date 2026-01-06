# Classifiers

Retries are only safe when they respect protocol/domain semantics.

Classifiers implement:

```go
Classify(value any, err error) classify.Outcome
```

The executor records `Outcome` on every attempt, and uses it to decide whether to retry, stop, or abort immediately.
<!-- Claim-ID: CLM-023 -->

## Built-ins

Core built-ins include:

- `classify.AutoClassifier` (default): Dispatches to `HTTPClassifier` when the error implements `HTTPError`, otherwise uses `AlwaysRetryOnError`. This works automatically with `recourse/integrations/http`, which returns errors implementing `HTTPError`.
  <!-- Claim-ID: CLM-005 -->
- `classify.ClassifierHTTP` (`"http"`): HTTP-aware decisions with idempotent-method rules; retries idempotent transport errors, 5xx, 408/429 (and configured extra 4xx), and honors `Retry-After` for backoff override.
  <!-- Claim-ID: CLM-006 -->
- `integrations/grpc.Classifier`: gRPC status-code aware decisions (non-gRPC errors delegate to `AutoClassifier`).
  <!-- Claim-ID: CLM-007 -->

Select a classifier by name via `policy.RetryPolicy.ClassifierName`.

## Safety: type mismatches

If a classifier expects a specific value/error shape and receives something else, it should fail loudly and safely (e.g., non-retryable with a clear reason), not “retry blindly”.
