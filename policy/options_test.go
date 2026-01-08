package policy

import (
	"testing"
	"time"
)

func TestNew_AppliesOptions(t *testing.T) {
	p := New("test.key",
		MaxAttempts(5),
		InitialBackoff(100*time.Millisecond),
		Classifier("custom"),
	)

	if p.Retry.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts 5, got %d", p.Retry.MaxAttempts)
	}
	if p.Retry.InitialBackoff != 100*time.Millisecond {
		t.Errorf("expected InitialBackoff 100ms, got %v", p.Retry.InitialBackoff)
	}
	if p.Retry.ClassifierName != "custom" {
		t.Errorf("expected ClassifierName 'custom', got %s", p.Retry.ClassifierName)
	}
	if p.Key.String() != "test.key" {
		t.Errorf("expected key 'test.key', got %s", p.Key.String())
	}
}

func TestNew_NormalizationFallback(t *testing.T) {

	invalidJitter := func(p *EffectivePolicy) {
		p.Retry.Jitter = JitterKind("invalid-jitter")
	}

	p := New("test.broken", invalidJitter)

	// Should fall back to default
	if p.Retry.MaxAttempts != 3 { // default
		t.Errorf("expected default MaxAttempts 3, got %d", p.Retry.MaxAttempts)
	}
	if p.Retry.Jitter != JitterNone { // default
		t.Errorf("expected default JitterNone, got %v", p.Retry.Jitter)
	}
}

func TestPresets_HTTPDefaults(t *testing.T) {
	p := New("test.http", HTTPDefaults())

	if p.Retry.ClassifierName != "http" {
		t.Errorf("expected http classifier, got %s", p.Retry.ClassifierName)
	}
	if p.Retry.BackoffMultiplier != 2.0 {
		t.Errorf("expected multiplier 2.0, got %f", p.Retry.BackoffMultiplier)
	}
	if p.Retry.MaxAttempts != 3 {
		t.Errorf("expected 3 attempts, got %d", p.Retry.MaxAttempts)
	}
}

func TestPresets_LowLatencyDefaults(t *testing.T) {
	p := New("test.fast", LowLatencyDefaults())

	if !p.Hedge.Enabled {
		t.Error("expected hedging enabled")
	}
	if p.Retry.MaxAttempts != 2 {
		t.Errorf("expected 2 attempts, got %d", p.Retry.MaxAttempts)
	}
}

func TestExponentialBackoff(t *testing.T) {
	p := New("test.exp", ExponentialBackoff(50*time.Millisecond, 5*time.Second))

	if p.Retry.InitialBackoff != 50*time.Millisecond {
		t.Errorf("expected initial 50ms, got %v", p.Retry.InitialBackoff)
	}
	if p.Retry.MaxBackoff != 5*time.Second {
		t.Errorf("expected max 5s, got %v", p.Retry.MaxBackoff)
	}
	if p.Retry.Jitter != JitterEqual {
		t.Errorf("expected JitterEqual, got %v", p.Retry.Jitter)
	}
}

func TestOptionSetters(t *testing.T) {
	p := New("test.opts",
		MaxAttempts(4),
		InitialBackoff(100*time.Millisecond),
		MaxBackoff(2*time.Second),
		BackoffMultiplier(3),
		Jitter(JitterFull),
		PerAttemptTimeout(5*time.Second),
		OverallTimeout(20*time.Second),
		Classifier("custom"),
		BudgetWithCost("budget", 3),
		PolicyID("policy-1"),
		HedgeMaxAttempts(2),
		HedgeDelay(150*time.Millisecond),
		HedgeTrigger("p95"),
		HedgeBudget("hedge-budget"),
		HedgeCancelOnTerminal(true),
	)

	if p.Retry.MaxAttempts != 4 || p.Retry.InitialBackoff != 100*time.Millisecond {
		t.Fatalf("unexpected retry settings: %+v", p.Retry)
	}
	if p.Retry.MaxBackoff != 2*time.Second || p.Retry.BackoffMultiplier != 3 {
		t.Fatalf("unexpected backoff: %+v", p.Retry)
	}
	if p.Retry.Jitter != JitterFull {
		t.Fatalf("jitter=%v, want %v", p.Retry.Jitter, JitterFull)
	}
	if p.Retry.TimeoutPerAttempt != 5*time.Second || p.Retry.OverallTimeout != 20*time.Second {
		t.Fatalf("unexpected timeouts: %+v", p.Retry)
	}
	if p.Retry.ClassifierName != "custom" {
		t.Fatalf("classifier=%q, want custom", p.Retry.ClassifierName)
	}
	if p.Retry.Budget.Name != "budget" || p.Retry.Budget.Cost != 3 {
		t.Fatalf("budget=%+v, want name=budget cost=3", p.Retry.Budget)
	}
	if p.ID != "policy-1" {
		t.Fatalf("id=%q, want policy-1", p.ID)
	}
	if !p.Hedge.Enabled {
		t.Fatalf("expected hedging enabled")
	}
	if p.Hedge.MaxHedges != 2 || p.Hedge.HedgeDelay != 150*time.Millisecond {
		t.Fatalf("unexpected hedge settings: %+v", p.Hedge)
	}
	if p.Hedge.TriggerName != "p95" {
		t.Fatalf("trigger=%q, want p95", p.Hedge.TriggerName)
	}
	if p.Hedge.Budget.Name != "hedge-budget" || p.Hedge.Budget.Cost != 1 {
		t.Fatalf("hedge budget=%+v, want name=hedge-budget cost=1", p.Hedge.Budget)
	}
	if !p.Hedge.CancelOnFirstTerminal {
		t.Fatalf("expected cancel-on-terminal enabled")
	}
}

func TestEnableHedgingDefaults(t *testing.T) {
	p := New("test.hedge", EnableHedging())
	if !p.Hedge.Enabled {
		t.Fatalf("expected hedging enabled")
	}
	if p.Hedge.MaxHedges != 2 {
		t.Fatalf("maxHedges=%d, want 2", p.Hedge.MaxHedges)
	}
	if p.Hedge.HedgeDelay != 200*time.Millisecond {
		t.Fatalf("hedgeDelay=%v, want 200ms", p.Hedge.HedgeDelay)
	}
}

func TestConstantBackoff(t *testing.T) {
	p := New("test.constant", ConstantBackoff(250*time.Millisecond))
	if p.Retry.InitialBackoff != 250*time.Millisecond || p.Retry.MaxBackoff != 250*time.Millisecond {
		t.Fatalf("unexpected backoff: %+v", p.Retry)
	}
	if p.Retry.BackoffMultiplier != 1.0 {
		t.Fatalf("multiplier=%v, want 1.0", p.Retry.BackoffMultiplier)
	}
	if p.Retry.Jitter != JitterNone {
		t.Fatalf("jitter=%v, want %v", p.Retry.Jitter, JitterNone)
	}
}

func TestPresets_DatabaseAndBackgroundDefaults(t *testing.T) {
	db := New("test.db", DatabaseDefaults())
	if db.Retry.MaxAttempts != 3 || db.Retry.MaxBackoff != 5*time.Second {
		t.Fatalf("unexpected db defaults: %+v", db.Retry)
	}

	bg := New("test.bg", BackgroundJobDefaults())
	if bg.Retry.MaxAttempts != 5 || bg.Retry.OverallTimeout != 5*time.Minute {
		t.Fatalf("unexpected bg defaults: %+v", bg.Retry)
	}
}

func TestBackoffOption(t *testing.T) {
	p := New("test.backoff", Backoff(20*time.Millisecond, 2*time.Second, 4))
	if p.Retry.InitialBackoff != 20*time.Millisecond {
		t.Fatalf("initial=%v, want 20ms", p.Retry.InitialBackoff)
	}
	if p.Retry.MaxBackoff != 2*time.Second {
		t.Fatalf("max=%v, want 2s", p.Retry.MaxBackoff)
	}
	if p.Retry.BackoffMultiplier != 4 {
		t.Fatalf("multiplier=%v, want 4", p.Retry.BackoffMultiplier)
	}
}

func TestBudgetOption(t *testing.T) {
	p := New("test.budget", Budget("budget"))
	if p.Retry.Budget.Name != "budget" || p.Retry.Budget.Cost != 1 {
		t.Fatalf("budget=%+v, want name=budget cost=1", p.Retry.Budget)
	}
}
