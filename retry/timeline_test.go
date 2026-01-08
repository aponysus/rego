package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aponysus/recourse/budget"
	"github.com/aponysus/recourse/circuit"
	"github.com/aponysus/recourse/controlplane"
	"github.com/aponysus/recourse/policy"
)

func TestDoValueWithTimeline_CircuitOpenShortCircuits(t *testing.T) {
	key := policy.PolicyKey{Name: "circuit"}
	cfg := policy.CircuitPolicy{Enabled: true, Threshold: 1, Cooldown: time.Minute}

	circuits := circuit.NewRegistry()
	cb := circuits.Get(key, cfg)
	if cb == nil {
		t.Fatal("expected circuit breaker")
	}
	cb.RecordFailure(context.Background())
	if cb.State() != circuit.StateOpen {
		t.Fatalf("expected open circuit, got %v", cb.State())
	}

	exec := NewExecutorFromOptions(ExecutorOptions{
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: {
					Key: key,
					Retry: policy.RetryPolicy{
						MaxAttempts: 2,
					},
					Circuit: cfg,
				},
			},
		},
		Circuits: circuits,
	})

	called := false
	_, tl, err := doValueWithTimeline(context.Background(), exec, key, func(context.Context) (int, error) {
		called = true
		return 1, nil
	})

	if called {
		t.Fatal("op should not be called when circuit is open")
	}

	var coe CircuitOpenError
	if !errors.As(err, &coe) {
		t.Fatalf("expected CircuitOpenError, got %v", err)
	}
	if tl.Attributes["circuit_state"] != "open" {
		t.Fatalf("circuit_state=%q, want %q", tl.Attributes["circuit_state"], "open")
	}
	if len(tl.Attempts) != 0 {
		t.Fatalf("attempts=%d, want 0", len(tl.Attempts))
	}
}

func TestDoValueWithTimeline_BudgetDeniedReturnsPreviousError(t *testing.T) {
	key := policy.PolicyKey{Name: "budget"}
	budgets := budget.NewRegistry()
	budgets.MustRegister("b", denySecondAttemptBudget{})

	exec := NewExecutorFromOptions(ExecutorOptions{
		Budgets: budgets,
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: {
					Key: key,
					Retry: policy.RetryPolicy{
						MaxAttempts: 2,
						Budget:      policy.BudgetRef{Name: "b", Cost: 1},
					},
				},
			},
		},
	})
	exec.sleep = func(context.Context, time.Duration) error { return nil }

	firstErr := errors.New("first failure")
	calls := 0
	_, tl, err := doValueWithTimeline(context.Background(), exec, key, func(context.Context) (int, error) {
		calls++
		return 0, firstErr
	})

	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
	if err != firstErr {
		t.Fatalf("err=%v, want %v", err, firstErr)
	}
	if len(tl.Attempts) != 2 {
		t.Fatalf("attempts=%d, want 2", len(tl.Attempts))
	}
	if tl.Attempts[1].BudgetAllowed {
		t.Fatalf("attempt[1].BudgetAllowed=true, want false")
	}
	if tl.Attempts[1].BudgetReason != budget.ReasonBudgetDenied {
		t.Fatalf("attempt[1].BudgetReason=%q, want %q", tl.Attempts[1].BudgetReason, budget.ReasonBudgetDenied)
	}
}

func TestDoValueWithTimeline_SleepErrorStopsRetry(t *testing.T) {
	key := policy.PolicyKey{Name: "sleep"}
	exec := NewExecutorFromOptions(ExecutorOptions{
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: {
					Key: key,
					Retry: policy.RetryPolicy{
						MaxAttempts:    2,
						InitialBackoff: 10 * time.Millisecond,
						Jitter:         policy.JitterNone,
					},
				},
			},
		},
	})

	sleepErr := errors.New("sleep failed")
	exec.sleep = func(context.Context, time.Duration) error { return sleepErr }

	calls := 0
	_, tl, err := doValueWithTimeline(context.Background(), exec, key, func(context.Context) (int, error) {
		calls++
		return 0, errors.New("retryable")
	})
	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
	if err != sleepErr {
		t.Fatalf("err=%v, want %v", err, sleepErr)
	}
	if len(tl.Attempts) != 1 {
		t.Fatalf("attempts=%d, want 1", len(tl.Attempts))
	}
}

func TestDoValue_FallsBackToTimelineWhenCircuitEnabled(t *testing.T) {
	key := policy.PolicyKey{Name: "fallback"}
	exec := NewExecutorFromOptions(ExecutorOptions{
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: {
					Key: key,
					Retry: policy.RetryPolicy{
						MaxAttempts: 1,
					},
					Circuit: policy.CircuitPolicy{
						Enabled:   true,
						Threshold: 2,
						Cooldown:  time.Second,
					},
				},
			},
		},
	})

	calls := 0
	_, err := DoValue(context.Background(), exec, key, func(context.Context) (int, error) {
		calls++
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
}
