package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aponysus/recourse/budget"
	"github.com/aponysus/recourse/controlplane"
	"github.com/aponysus/recourse/policy"
)

type denySecondAttemptBudget struct{}

func (denySecondAttemptBudget) AllowAttempt(_ context.Context, _ policy.PolicyKey, attemptIdx int, _ budget.AttemptKind, _ policy.BudgetRef) budget.Decision {
	if attemptIdx == 1 {
		return budget.Decision{Allowed: false, Reason: budget.ReasonBudgetDenied}
	}
	return budget.Decision{Allowed: true, Reason: "allowed"}
}

type countingReleaseBudget struct {
	allowCalls int32
	releases   int32
}

func (b *countingReleaseBudget) AllowAttempt(_ context.Context, _ policy.PolicyKey, _ int, _ budget.AttemptKind, _ policy.BudgetRef) budget.Decision {
	atomic.AddInt32(&b.allowCalls, 1)
	return budget.Decision{
		Allowed: true,
		Reason:  "allowed",
		Release: func() {
			atomic.AddInt32(&b.releases, 1)
		},
	}
}

func TestExecutor_BudgetDeniesSecondAttempt_StopsRetryAndReturnsLastErr(t *testing.T) {
	key := policy.PolicyKey{Name: "x"}

	budgets := budget.NewRegistry()
	budgets.Register("b", denySecondAttemptBudget{})

	exec := NewExecutor(ExecutorOptions{
		Budgets: budgets,
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: {
					Key: key,
					Retry: policy.RetryPolicy{
						MaxAttempts: 3,
						Budget:      policy.BudgetRef{Name: "b", Cost: 1},
					},
				},
			},
		},
	})
	exec.sleep = func(context.Context, time.Duration) error { return nil }

	calls := 0
	opErr := errors.New("first attempt error")
	_, tl, err := DoValueWithTimeline[int](context.Background(), exec, key, func(context.Context) (int, error) {
		calls++
		return 0, opErr
	})

	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
	if err != opErr {
		t.Fatalf("err=%v, want %v", err, opErr)
	}
	if tl.FinalErr != opErr {
		t.Fatalf("tl.FinalErr=%v, want %v", tl.FinalErr, opErr)
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
	if tl.Attempts[1].Err != nil {
		t.Fatalf("attempt[1].Err=%v, want nil (op not called)", tl.Attempts[1].Err)
	}
}

func TestExecutor_MissingBudgetName_AllowsAttemptsAndRecordsReason(t *testing.T) {
	key := policy.PolicyKey{Name: "x"}

	exec := NewExecutor(ExecutorOptions{
		Budgets: budget.NewRegistry(), // empty registry
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: {
					Key: key,
					Retry: policy.RetryPolicy{
						MaxAttempts: 2,
						Budget:      policy.BudgetRef{Name: "missing", Cost: 1},
					},
				},
			},
		},
	})
	exec.sleep = func(context.Context, time.Duration) error { return nil }

	calls := 0
	_, tl, err := DoValueWithTimeline[int](context.Background(), exec, key, func(context.Context) (int, error) {
		calls++
		return 0, errors.New("nope")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2", calls)
	}
	if len(tl.Attempts) != 2 {
		t.Fatalf("attempts=%d, want 2", len(tl.Attempts))
	}
	for i, rec := range tl.Attempts {
		if !rec.BudgetAllowed {
			t.Fatalf("attempt[%d].BudgetAllowed=false, want true", i)
		}
		if rec.BudgetReason != budget.ReasonBudgetNotFound {
			t.Fatalf("attempt[%d].BudgetReason=%q, want %q", i, rec.BudgetReason, budget.ReasonBudgetNotFound)
		}
	}
}

func TestExecutor_BudgetRelease_CalledOncePerAllowedAttempt(t *testing.T) {
	key := policy.PolicyKey{Name: "x"}

	cb := &countingReleaseBudget{}
	budgets := budget.NewRegistry()
	budgets.Register("b", cb)

	exec := NewExecutor(ExecutorOptions{
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

	calls := 0
	_, _, err := DoValueWithTimeline[int](context.Background(), exec, key, func(context.Context) (int, error) {
		calls++
		if calls == 1 {
			return 0, errors.New("nope")
		}
		return 42, nil
	})
	if err != nil {
		t.Fatalf("err=%v, want nil", err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2", calls)
	}
	if got, want := atomic.LoadInt32(&cb.allowCalls), int32(2); got != want {
		t.Fatalf("allowCalls=%d, want %d", got, want)
	}
	if got, want := atomic.LoadInt32(&cb.releases), int32(2); got != want {
		t.Fatalf("releases=%d, want %d", got, want)
	}
}
