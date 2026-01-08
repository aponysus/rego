package controlplane

import (
	"context"
	"testing"

	"github.com/aponysus/recourse/policy"
)

func TestStaticProvider_ReturnsPolicyAndSetsSource(t *testing.T) {
	key := policy.ParseKey("svc.method")
	provider := &StaticProvider{
		Policies: map[policy.PolicyKey]policy.EffectivePolicy{
			key: {Retry: policy.RetryPolicy{MaxAttempts: 4}},
		},
	}

	pol, err := provider.GetEffectivePolicy(context.Background(), key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pol.Key != key {
		t.Fatalf("key=%v, want %v", pol.Key, key)
	}
	if pol.Meta.Source != policy.PolicySourceStatic {
		t.Fatalf("source=%v, want %v", pol.Meta.Source, policy.PolicySourceStatic)
	}
	if pol.Retry.MaxAttempts != 4 {
		t.Fatalf("maxAttempts=%d, want 4", pol.Retry.MaxAttempts)
	}
}

func TestStaticProvider_PreservesSource(t *testing.T) {
	key := policy.ParseKey("svc.method")
	provider := &StaticProvider{
		Policies: map[policy.PolicyKey]policy.EffectivePolicy{
			key: {Retry: policy.RetryPolicy{MaxAttempts: 4}, Meta: policy.Metadata{Source: policy.PolicySourceRemote}},
		},
	}

	pol, err := provider.GetEffectivePolicy(context.Background(), key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pol.Meta.Source != policy.PolicySourceRemote {
		t.Fatalf("source=%v, want %v", pol.Meta.Source, policy.PolicySourceRemote)
	}
}

func TestStaticProvider_UsesDefault(t *testing.T) {
	key := policy.ParseKey("svc.missing")
	provider := &StaticProvider{
		Default: policy.EffectivePolicy{Retry: policy.RetryPolicy{MaxAttempts: 5}},
	}

	pol, err := provider.GetEffectivePolicy(context.Background(), key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pol.Key != key {
		t.Fatalf("key=%v, want %v", pol.Key, key)
	}
	if pol.Meta.Source != policy.PolicySourceStatic {
		t.Fatalf("source=%v, want %v", pol.Meta.Source, policy.PolicySourceStatic)
	}
	if pol.Retry.MaxAttempts != 5 {
		t.Fatalf("maxAttempts=%d, want 5", pol.Retry.MaxAttempts)
	}
}

func TestStaticProvider_FallbackToDefaultPolicy(t *testing.T) {
	key := policy.ParseKey("svc.default")
	var provider *StaticProvider

	pol, err := provider.GetEffectivePolicy(context.Background(), key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pol.Key != key {
		t.Fatalf("key=%v, want %v", pol.Key, key)
	}
	if pol.Meta.Source != policy.PolicySourceDefault {
		t.Fatalf("source=%v, want %v", pol.Meta.Source, policy.PolicySourceDefault)
	}
}

func TestIsZeroEffectivePolicy(t *testing.T) {
	if !isZeroEffectivePolicy(policy.EffectivePolicy{}) {
		t.Fatal("expected zero policy")
	}

	nonZero := policy.EffectivePolicy{
		Retry: policy.RetryPolicy{MaxAttempts: 1},
	}
	if isZeroEffectivePolicy(nonZero) {
		t.Fatal("expected non-zero policy")
	}
}
