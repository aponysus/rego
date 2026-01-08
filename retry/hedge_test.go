package retry

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aponysus/recourse/hedge"
	"github.com/aponysus/recourse/observe"
	"github.com/aponysus/recourse/policy"
)

func TestExecutor_Hedge_PrimaryWins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	key := policy.ParseKey("test.hedge.primary")
	pol := policy.EffectivePolicy{
		Key: key,
		Retry: policy.RetryPolicy{
			MaxAttempts: 1,
		},
		Hedge: policy.HedgePolicy{
			Enabled:     true,
			MaxHedges:   1,
			HedgeDelay:  0,
			TriggerName: "immediate",
		},
	}
	exec := newTestExecutor(t, key, pol)
	setImmediateTrigger(exec)

	ctx, capture := observe.RecordTimeline(context.Background())

	errHedgeNotStarted := errors.New("hedge did not start")
	hedgeStarted := make(chan struct{})
	var hedgeOnce sync.Once

	val, err := DoValue[string](ctx, exec, key, func(ctx context.Context) (string, error) {
		info, _ := observe.AttemptFromContext(ctx)
		if info.IsHedge {
			hedgeOnce.Do(func() { close(hedgeStarted) })
			<-ctx.Done()
			return "", ctx.Err()
		}
		if !waitForSignal(hedgeStarted) {
			return "", errHedgeNotStarted
		}
		return "ok", nil
	})

	if errors.Is(err, errHedgeNotStarted) {
		t.Fatal("expected hedge attempt to start")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "ok" {
		t.Errorf("got %v, want ok", val)
	}

	tl := capture.Timeline()

	count := len(tl.Attempts)
	if count < 1 {
		t.Fatalf("expected at least 1 attempt")
	}
}

func TestExecutor_Hedge_HedgeWins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	key := policy.ParseKey("test.hedge.secondary")
	pol := policy.EffectivePolicy{
		Key: key,
		Retry: policy.RetryPolicy{
			MaxAttempts: 1,
		},
		Hedge: policy.HedgePolicy{
			Enabled:     true,
			MaxHedges:   1,
			HedgeDelay:  0,
			TriggerName: "immediate",
		},
	}
	exec := newTestExecutor(t, key, pol)
	setImmediateTrigger(exec)

	ctx, capture := observe.RecordTimeline(context.Background())
	primaryDone := make(chan struct{})
	primaryStarted := make(chan struct{})
	hedgeStarted := make(chan struct{})
	var hedgeOnce sync.Once
	errHedgeNotStarted := errors.New("hedge did not start")

	val, err := DoValue[string](ctx, exec, key, func(ctx context.Context) (string, error) {
		info, _ := observe.AttemptFromContext(ctx)
		if !info.IsHedge {
			close(primaryStarted)
			defer close(primaryDone)
			if !waitForSignal(hedgeStarted) {
				return "", errHedgeNotStarted
			}
			<-ctx.Done()
			return "primary", ctx.Err()
		}
		hedgeOnce.Do(func() { close(hedgeStarted) })
		if !waitForSignal(primaryStarted) {
			return "", errHedgeNotStarted
		}
		return "hedge", nil
	})

	if errors.Is(err, errHedgeNotStarted) {
		t.Fatal("expected hedge attempt to start")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hedge" {
		t.Errorf("got %v, want hedge", val)
	}

	// Must wait for primary to finish recording
	if !waitForSignal(primaryDone) {
		t.Fatal("primary did not finish")
	}

	tl := capture.Timeline()
	// Should show at least Hedge attempting.
	// Primary attempt might not be recorded if it finishes after return (due to async cancel).
	if len(tl.Attempts) < 1 {
		t.Errorf("expected at least 1 attempt, got %d", len(tl.Attempts))
	}

	// Verify one is hedge
	hasHedge := false
	for _, a := range tl.Attempts {
		if a.IsHedge {
			hasHedge = true
		}
	}
	if !hasHedge {
		t.Error("expected at least one hedge attempt")
	}
}

func TestExecutor_Hedge_RetryAndHedge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	key := policy.ParseKey("test.hedge.retry")
	pol := policy.EffectivePolicy{
		Key: key,
		Retry: policy.RetryPolicy{
			MaxAttempts:    2,
			InitialBackoff: 0,
		},
		Hedge: policy.HedgePolicy{
			Enabled:     true,
			MaxHedges:   1,
			HedgeDelay:  0,
			TriggerName: "immediate",
		},
	}
	exec := newTestExecutor(t, key, pol)
	setImmediateTrigger(exec)

	ctx, capture := observe.RecordTimeline(context.Background())
	errHedgeNotStarted := errors.New("hedge did not start")
	hedgeStarted := []chan struct{}{make(chan struct{}), make(chan struct{})}
	primaryStarted := []chan struct{}{make(chan struct{}), make(chan struct{})}
	hedgeOnce := make([]sync.Once, len(hedgeStarted))
	var missingHedge atomic.Bool

	_, err := DoValue[string](ctx, exec, key, func(ctx context.Context) (string, error) {
		info, _ := observe.AttemptFromContext(ctx)
		idx := info.RetryIndex
		if idx < 0 || idx >= len(hedgeStarted) {
			return "", errors.New("unexpected retry index")
		}
		if info.IsHedge {
			hedgeOnce[idx].Do(func() { close(hedgeStarted[idx]) })
			if !waitForSignal(primaryStarted[idx]) {
				missingHedge.Store(true)
				return "", errHedgeNotStarted
			}
			return "", context.DeadlineExceeded
		}
		close(primaryStarted[idx])
		if !waitForSignal(hedgeStarted[idx]) {
			missingHedge.Store(true)
			return "", errHedgeNotStarted
		}
		return "", context.DeadlineExceeded
	})

	if missingHedge.Load() || errors.Is(err, errHedgeNotStarted) {
		t.Fatal("expected hedge attempts to start")
	}
	if err == nil {
		t.Fatal("expected error")
	}

	tl := capture.Timeline()
	// Retry 0: Primary + Hedge.
	// Retry 1: Primary + Hedge.
	// Total 4.
	if len(tl.Attempts) != 4 {
		t.Errorf("expected 4 attempts, got %d", len(tl.Attempts))
	}
}

type immediateTrigger struct{}

func (immediateTrigger) ShouldSpawnHedge(state hedge.HedgeState) (bool, time.Duration) {
	if state.AttemptsLaunched >= 1+state.MaxHedges {
		return false, 0
	}
	return true, 0
}

func setImmediateTrigger(exec *Executor) {
	triggers := hedge.NewRegistry()
	triggers.Register("immediate", immediateTrigger{})
	exec.triggers = triggers
}
