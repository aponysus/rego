# Why Naive Resilience Logic Fails (and How recourse Tries to Fix It)

Retry logic is one of those things that always feels easy - right up until the day it isn’t.

Most of the time it gets added under pressure:

* “Just retry a couple times.”
* “Add exponential backoff.”
* “Try it again if it fails.”

It works in dev. It mostly works in staging. Then production hits: a dependency gets flaky, latency spikes, or a downstream starts rate-limiting…and suddenly those “simple retries” turn a small blip into a real incident.

recourse is my attempt to design resilience the way I wish it were treated more often in Go codebases: **consistent**, **bounded**, and **observable** - with explicit backpressure so retries don’t become self-inflicted outages.

---

## 1. A simple retry loop isn’t enough

The common Go retry loop looks roughly like this:

```go
var lastErr error
for attempt := 0; attempt < 3; attempt++ {
    if err := callDependency(ctx); err == nil {
        return nil
    } else {
        lastErr = err
    }
    time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
}
return lastErr
```

The loop itself isn’t the real problem. The problem is everything you *eventually* need around it:

* Is the failure retryable (HTTP 500 vs 404 vs 429)?
* Are you respecting **per-attempt** vs **overall** timeouts?
* Do you have explicit **backpressure** so an outage doesn’t become a retry storm?
* When it’s 3am, can you answer: **what happened on each attempt, and why?**

recourse exists because production-grade retry behavior is not “3 tries and a sleep.” It’s a set of operational guarantees.

---

## 2. Policy-driven resilience: move the *envelope* out of the call site

Most retry libraries are “mechanism-first”: you configure attempt counts and delays right where you call the dependency.

recourse flips that around: the call site supplies a **policy key**, and the retry envelope is defined centrally by policy.

```go
user, err := recourse.DoValue[User](ctx, "user-service.GetUser", func(ctx context.Context) (User, error) {
    return client.GetUser(ctx, userID)
})
```

The policy tied to `"user-service.GetUser"` can control:

* max attempts
* backoff + jitter
* timeouts (per attempt and overall)
* which classifier interprets errors/results
* budgets (backpressure)
* hedging configuration
* circuit breaking configuration

This is the core design bet: **call sites should be boring**. Resilience behavior should be consistent, tunable, and auditable without re-implementing retry loops everywhere.

---

## 3. The “key” must be low-cardinality (or you’ll regret it)

Once you make a key the unit of control, it becomes the unit of:

* policy selection
* observability dimensions
* caches / trackers (including latency tracking for hedging)

That means keys **cannot** include things like user IDs, request IDs, or per-entity routing details.

Good:

* `"payments.Charge"`
* `"user-service.GetUser"`

Bad:

* `"GET /users/123"`
* `"user-service.GetUser?user_id=123"`

This isn’t pedantry. High-cardinality keys lead to unbounded memory growth, unusable metrics, and confusing operational data. ADR-001 is explicit about this because the whole architecture assumes keys are safe to aggregate on.

---

## 4. Guardrails aren’t optional: bounded envelopes + policy normalization

One reason retries go off the rails is that configuration drifts into unsafe territory:

* too many attempts
* tiny timeouts that busy-loop
* backoff values that don’t make sense
* invalid enum values

recourse treats this as a first-class failure mode. Every `EffectivePolicy` is normalized/clamped via `EffectivePolicy.Normalize()`:

* fill missing values with safe defaults
* clamp values to documented safe ranges
* reject invalid enum values
* record normalization metadata for observability

This is a design principle I care about a lot: **you should be able to load policies dynamically without trusting them blindly**.

And importantly: if policy resolution fails, behavior is explicit. By default, recourse fails closed (`FailureDeny`) with `retry.ErrNoPolicy`, but you can opt into “single attempt” (`FailureAllow`) or a safe fallback policy (`FailureFallback`).

---

## 5. Not every failure is retryable: classifiers, not heuristics

Naive retries treat all failures the same. Real systems don’t.

recourse makes this explicit with **classifiers**: a classifier maps `(value, err)` into an `Outcome`, and the executor uses that outcome to decide whether to retry, stop, or abort.

Built-ins include:

* `classify.AutoClassifier` (default): dispatches based on error type (e.g., HTTP-aware when using the HTTP integration)
* `classify.ClassifierHTTP`: understands HTTP semantics (5xx vs 404 vs 429 + `Retry-After`)
* a gRPC classifier in the gRPC integration module that interprets gRPC status codes

A key safety rule here: **type mismatches should fail loudly and safely**, not degrade into “retry blindly.” If a classifier expects one shape and receives another, the conservative answer is non-retryable/abort with a clear reason.

---

## 6. Observability-first: retries are only “safe” when they’re explainable

Retries hide behavior unless you surface it deliberately.

recourse gives you two complementary observability paths:

### A structured per-call timeline

You can capture an `observe.Timeline` on demand:

```go
ctx, capture := observe.RecordTimeline(ctx)

user, err := recourse.DoValue(ctx, "user-service.GetUser", op)

// After the call:
tl := capture.Timeline()
for _, a := range tl.Attempts {
    // a.Outcome, a.Err, a.Backoff, a.IsHedge, a.BudgetAllowed, ...
}
```

The timeline records per-attempt timings, outcomes, errors, backoff decisions, and budget gating, plus call-level attributes (like policy source and whether normalization/clamping happened).

### Streaming observer hooks

For logs/metrics/tracing integrations, you can implement `observe.Observer`. The observer receives lifecycle events (start/success/failure), attempt events, hedge spawn events, and budget decision events with standardized reasons like `"budget_denied"` or `"circuit_open"`.

This is the point: **recourse doesn’t just retry - it tells you exactly what it did.**

---

## 7. Backpressure: budgets make retries outage-safe

Retries and hedges multiply load. If a dependency is struggling, naive retries can turn “slow” into “down.”

Budgets in recourse provide a per-attempt gate:

* allow the attempt
* deny the attempt (and record why)
* optionally return a release handle for reservation-style resources

There are built-ins like:

* `budget.UnlimitedBudget` (always allows)
* `budget.TokenBucketBudget` (capacity + refill rate)

And the failure modes are explicit:

* empty budget name → allowed with reason `"no_budget"`
* missing registry / missing budget / nil budget → controlled by `MissingBudgetMode` (default is fail-closed) and recorded as `"budget_registry_nil"`, `"budget_not_found"`, or `"budget_nil"`

This is one of the biggest philosophical differences from “retry helpers”: **backpressure isn’t an afterthought; it’s part of the retry contract.**

---

## 8. Tail latency is also reliability: hedging

Even if your average latency looks fine, p99/p999 spikes can dominate user experience and upstream timeouts.

recourse supports hedging: starting a second (or third) attempt while the first is still in flight, racing them so the first success wins.

There are two modes:

* **Fixed delay**: spawn a hedge after N milliseconds
* **Latency-aware**: spawn a hedge dynamically based on recent latency stats (e.g. p99), using a per-key tracker (documented as a ring buffer)

The behavior is intentionally explicit:

* **Winner-takes-all**: first success cancels other in-flight attempts
* **Fail-fast option**: `CancelOnFirstTerminal` can stop the group on a non-retryable error; otherwise, other in-flight attempts can still “win”
* **Budgets**: hedges can have their own budget (`Hedge.Budget`) independent of the retry budget

Hedging is powerful, but it’s also a lever that can increase load. The reason it works in recourse is the combination of **budgets**, **deterministic group behavior**, and **first-class observability**.

---

## 9. When it’s failing, stop sending traffic: circuit breaking

Retries help when failures are transient. When a downstream is persistently failing, retries can just add fuel.

recourse includes a consecutive-failure circuit breaker with standard states:

* **Closed**: normal operation, record successes/failures
* **Open**: fail fast with `CircuitOpenError`
* **Half-open**: after cooldown, allow limited probes; success closes the circuit, failure re-opens it

One detail I care about: **circuit breaking is integrated with the rest of the loop**, not bolted on. For example, hedging is disabled in half-open probing to avoid hammering a recovering dependency.

---

## 10. Operate it like a system: remote configuration (with caching + negative caching)

If you buy the “policy-driven” model, the next step is obvious: update policies without redeploying.

recourse supports a `controlplane.RemoteProvider` that fetches policies from an external source and caches them. The docs call out:

* TTL caching for fetched policies
* negative caching for not-found policies (e.g. 404) to prevent hot-spotting on missing keys
* explicit fallback behavior via `MissingPolicyMode` when the source is unavailable

This is the operational through-line: **centralize control**, **avoid hammering your control plane**, and **make fallback behavior explicit**.

---

## 11. Integration philosophy: standard library first, dependencies opt-in

In Go, “resilience libraries” often become frameworks with heavy wrappers.

recourse explicitly tries not to do that:

* Integrations target standard interfaces (`net/http`, gRPC interceptors) rather than forcing custom clients
* The core library stays light; heavy dependencies (like gRPC) live in separate modules/packages
* Helpers handle correctness edge cases (like draining/closing response bodies on HTTP retries so connections can be reused)

For example, the HTTP integration (`integrations/http`) provides a `DoHTTP` helper that classifies HTTP status codes, respects `Retry-After` in relevant cases, and manages response bodies for failed attempts.

---

## 12. Getting started: three “depth levels”

One thing I wanted from the beginning was a library that you can adopt incrementally:

### Level 1: “Just do it” (facade)

```go
user, err := recourse.DoValue[User](ctx, "user-service.GetUser", op)
```

This is the lowest-friction entry point.

### Level 2: Make it explainable (timeline capture)

```go
ctx, capture := observe.RecordTimeline(ctx)
_, _ = recourse.DoValue(ctx, "user-service.GetUser", op)
tl := capture.Timeline()
_ = tl
```

Timelines are the fastest way to debug real retry behavior.

### Level 3: Own the behavior (explicit executor + registries)

When you want to standardize defaults (HTTP classification, hedging triggers, budgets), use an explicit executor (or `NewDefaultExecutor`) and pass it around.

`NewDefaultExecutor` is documented as shipping with sensible defaults like `AutoClassifier`, an `"unlimited"` budget, and fixed/latency hedging triggers.

And when you need to integrate with HTTP or gRPC, use the drop-in integration packages rather than reinventing classification and lifecycle edge cases.

---

## Why this design matters

recourse isn’t trying to out-feature every retry library in the Go ecosystem.

It’s trying to make resilience behavior:

* **policy-driven** (keys select behavior, not ad-hoc config)
* **semantic** (classification rather than “retry on any error”)
* **bounded** (timeouts, caps, normalization/clamping)
* **outage-safe** (budgets/backpressure)
* **visible** (timelines + observer hooks)
* **composable** (hedging and circuit breaking integrated into one coherent execution model)


---
