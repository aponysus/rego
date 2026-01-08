package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aponysus/recourse/circuit"
	"github.com/aponysus/recourse/controlplane"
	"github.com/aponysus/recourse/policy"
)

func TestExecutor_CircuitBreaker_OpenOnFailures(t *testing.T) {
	key := policy.PolicyKey{Name: "circuit_test"}
	pol := policy.EffectivePolicy{
		Key: key,
		Retry: policy.RetryPolicy{
			MaxAttempts: 1,
		},
		Circuit: policy.CircuitPolicy{
			Enabled:   true,
			Threshold: 2,
			Cooldown:  100 * time.Millisecond,
		},
	}

	reg := circuit.NewRegistry()
	clock := &fakeClock{now: time.Unix(0, 0)}
	cb := reg.Get(key, pol.Circuit)
	breaker, ok := cb.(*circuit.ConsecutiveFailureBreaker)
	if !ok {
		t.Fatalf("expected *ConsecutiveFailureBreaker, got %T", cb)
	}
	breaker.SetClock(clock.Now)
	exec := NewExecutorFromOptions(ExecutorOptions{
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: pol,
			},
		},
		Circuits: reg,
	})

	// 1. First Failure
	_, err := DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
		return 0, errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if cb.State() != circuit.StateClosed {
		t.Errorf("expected Closed after 1 fail, got %v", cb.State())
	}

	// 2. Second Failure (Should Open)
	_, err = DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
		return 0, errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if cb.State() != circuit.StateOpen {
		t.Errorf("expected Open after 2 fails, got %v", cb.State())
	}

	// 3. Third Attempt (Should be rejected immediately)
	_, err = DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
		t.Fatal("should not execute")
		return 0, nil
	})

	// Check error type
	var circuitErr CircuitOpenError
	if !errors.As(err, &circuitErr) {
		t.Errorf("expected CircuitOpenError, got %v", err)
	}
	if circuitErr.State != circuit.StateOpen {
		t.Errorf("expected state Open, got %v", circuitErr.State)
	}

	// 4. Wait for Cooldown
	clock.Advance(pol.Circuit.Cooldown + time.Millisecond)

	// 5. Probe (Should succeed and Close)
	_, err = DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
		return 100, nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// 6. Next attempt (Should be Allowed/Closed)
	_, err = DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
		return 200, nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestExecutor_CircuitBreaker_DisablesHedging(t *testing.T) {
	key := policy.PolicyKey{Name: "hedge_circuit_test"}
	pol := policy.EffectivePolicy{
		Key: key,
		Retry: policy.RetryPolicy{
			MaxAttempts: 1,
		},
		Hedge: policy.HedgePolicy{
			Enabled:    true,
			MaxHedges:  2,
			HedgeDelay: 0,
		},
		Circuit: policy.CircuitPolicy{
			Enabled:   true,
			Threshold: 1,
			Cooldown:  100 * time.Millisecond,
		},
	}

	reg := circuit.NewRegistry()
	clock := &fakeClock{now: time.Unix(0, 0)}
	cb := reg.Get(key, pol.Circuit)
	breaker, ok := cb.(*circuit.ConsecutiveFailureBreaker)
	if !ok {
		t.Fatalf("expected *ConsecutiveFailureBreaker, got %T", cb)
	}
	breaker.SetClock(clock.Now)

	exec := NewExecutorFromOptions(ExecutorOptions{
		Provider: &controlplane.StaticProvider{
			Policies: map[policy.PolicyKey]policy.EffectivePolicy{
				key: pol,
			},
		},
		Circuits: reg,
	})

	// 1. Fail to Open Circuit
	DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
		return 0, errors.New("fail")
	})

	// 2. Wait for Cooldown -> Half-Open
	clock.Advance(pol.Circuit.Cooldown + time.Millisecond)

	// 3. Probe with Hedging Configured
	// Verify that ONLY PRIMARY is launched (hedging disabled)
	var attempts atomic.Int32
	errHedgeExecuted := errors.New("hedge executed in half-open")
	_, err := DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
		attempts.Add(1)
		if spinWait(func() bool { return attempts.Load() > 1 }) {
			return 0, errHedgeExecuted
		}
		return 0, errors.New("fail") // Fail probe
	})

	// Should fail -> Open again
	if errors.Is(err, errHedgeExecuted) {
		t.Fatal("hedge should be disabled in half-open state")
	}
	if err == nil {
		t.Fatal("expected fail")
	}

	// Attempts should be 1 (Primary only)
	if n := attempts.Load(); n != 1 {
		t.Errorf("expected 1 attempt (hedging disabled), got %d. Did hedging trigger in Half-Open?", n)
	}
}

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	return f.now
}

func (f *fakeClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}
