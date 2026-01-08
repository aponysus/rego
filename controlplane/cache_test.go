package controlplane

import (
	"testing"
	"time"

	"github.com/aponysus/recourse/policy"
)

func TestPolicyCache_SetGetAndInvalidate(t *testing.T) {
	cache := NewPolicyCache()
	key := policy.ParseKey("svc.method")
	pol := policy.EffectivePolicy{Retry: policy.RetryPolicy{MaxAttempts: 2}}

	cache.Set(key, pol, 50*time.Millisecond)
	got, found, negative := cache.Get(key)
	if !found || negative {
		t.Fatalf("expected positive cache hit")
	}
	if got.Retry.MaxAttempts != 2 {
		t.Fatalf("got MaxAttempts=%d, want 2", got.Retry.MaxAttempts)
	}

	cache.SetMissing(key, 50*time.Millisecond)
	_, found, negative = cache.Get(key)
	if !found || !negative {
		t.Fatalf("expected negative cache hit")
	}

	cache.Invalidate(key)
	_, found, negative = cache.Get(key)
	if found || negative {
		t.Fatalf("expected cache miss after invalidate")
	}
}

func TestPolicyCache_Expiry(t *testing.T) {
	cache := NewPolicyCache()
	key := policy.ParseKey("svc.expire")
	pol := policy.EffectivePolicy{}

	cache.Set(key, pol, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	_, found, negative := cache.Get(key)
	if found || negative {
		t.Fatalf("expected expired cache entry to miss")
	}
}
