package circuit

import (
	"context"
	"testing"
	"time"

	"github.com/aponysus/recourse/policy"
)

func TestRegistry_DisabledCircuitReturnsNil(t *testing.T) {
	reg := NewRegistry()
	key := policy.ParseKey("svc.Method")
	cb := reg.Get(key, policy.CircuitPolicy{Enabled: false})
	if cb != nil {
		t.Fatal("expected nil circuit breaker when disabled")
	}
}

func TestRegistry_ReusesBreakerPerKey(t *testing.T) {
	reg := NewRegistry()
	cfg := policy.CircuitPolicy{Enabled: true, Threshold: 2, Cooldown: 5 * time.Millisecond}

	key := policy.ParseKey("svc.Method")
	cb1 := reg.Get(key, cfg)
	cb2 := reg.Get(key, cfg)
	if cb1 == nil || cb2 == nil {
		t.Fatal("expected non-nil breaker")
	}
	if cb1 != cb2 {
		t.Fatal("expected same breaker for the same key")
	}

	other := policy.ParseKey("svc.Other")
	cb3 := reg.Get(other, cfg)
	if cb3 == nil || cb3 == cb1 {
		t.Fatal("expected distinct breaker for different key")
	}

	ctx := context.Background()
	cb1.RecordFailure(ctx)
	if cb1.State() != StateClosed {
		t.Fatalf("expected closed after 1 failure, got %v", cb1.State())
	}
	cb1.RecordFailure(ctx)
	if cb1.State() != StateOpen {
		t.Fatalf("expected open after 2 failures, got %v", cb1.State())
	}
}
