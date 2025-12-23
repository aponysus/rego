# Comparison with other libraries

There are many excellent resilience libraries in the Go ecosystem. `recourse` builds on the lessons learned from them but takes a different architectural approach.

## Summary

| Library | Policy-Driven? | Observability | Scope | Best for... |
|---|---|---|---|---|
| **`recourse`** | ✅ (Keys) | ✅ (Timeline/Log) | Retries, Hedging, Circuit Breaking, Budgets | Production microservices requiring consistency & visibility. |
| **`cenkalti/backoff`** | ❌ (Manual) | ❌ (Manual) | Retries (Backoff only) | Single scripts, simple helpers where observability is optional. |
| **`avast/retry-go`** | ❌ (Functional) | ⚠️ (Last error only) | Retries | Ergonomic function wrapping for simple needs. |
| **`hystrix-go`** | ✅ (Command) | ✅ (Stream) | Circuit Breaking, Concurrency Limits | **Legacy/Archived**. Heavy "command" pattern usage. |
| **`sony/gobreaker`** | ⚠️ (Struct) | ⚠️ (State change) | Circuit Breaking | Standalone circuit breaking without retries. |

## Detailed Comparison

### vs. `cenkalti/backoff` & `avast/retry-go`

These libraries are **mechanism-first**: you configure *how* to retry (count, delay) at the call site.

```go
// avast/retry-go
retry.Do(
    func() error { ... },
    retry.Attempts(3), // Configuration live at call site
    retry.Delay(1 * time.Second),
)
```

**The Problem**:
1.  **Drift**: Service A retries 3 times, Service B retries 5 times.
2.  **Opacity**: When `Do()` returns an error, you don't know *what* happened (did it retry? how many times? why did it fail?).
3.  **Naive**: Usually just "retry on error", lacking protocol awareness (e.g., retrying 404s).

**Recourse Approach**:
You provide a **key**. Configuration is centralized (and can be updated remotely).

```go
// recourse
recourse.Do(ctx, "database.Query", func(ctx context.Context) error { ... })
```

And you get a **timeline**:
> "Attempt 1 failed (500), backed off 10ms. Attempt 2 failed (500), backed off 20ms. Attempt 3 (Hedged) succeeded."

---

### vs. `afex/hystrix-go`

Hystrix (Java) pioneered the "Command" pattern and Circuit Breaking. `hystrix-go` is the Go port.

**The Problem**:
1.  **Archived**: The repo is no longer maintained.
2.  **Heavy**: Requires wrapping code in `hystrix.Go("command_name", func() error { ... })`.
3.  **Concurrency Limits**: Relies heavily on semaphore isolation, which can be rigid/complex to tune compared to token buckets.

**Recourse Approach**:
`recourse` uses standard context-aware functions. Circuit breaking is just one "middleware" in the retry loop, seamlessly integrated with hedging and retries. Budgeting (backpressure) is handled via token buckets, which handle burstiness better than strict concurrency limits.

---

### vs. `sony/gobreaker`

`gobreaker` is an excellent, focused implementation of the Circuit Breaker pattern.

**The Problem**:
It *only* does circuit breaking. It doesn't handle retries, backoff, jitter, or hedging. You have to glue it together with other libraries manually.

**Recourse Approach**:
`recourse` integrates `Circuit Breaker` state into the retry loop.
*   If the breaker is open, we don't just "fail"; we record the specific reason (`"circuit_open"`) in the timeline.
*   If the breaker is half-open (probing), we automatically **disable hedging** to avoid hammering the recovering service.
