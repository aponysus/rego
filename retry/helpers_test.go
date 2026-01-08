package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aponysus/recourse/controlplane"
)

func TestFailureModeString(t *testing.T) {
	cases := []struct {
		mode FailureMode
		want string
	}{
		{mode: FailureDeny, want: "deny"},
		{mode: FailureAllow, want: "allow"},
		{mode: FailureFallback, want: "fallback"},
		{mode: FailureAllowUnsafe, want: "allow_unsafe"},
		{mode: FailureModeUnknown, want: "unknown"},
	}

	for _, tc := range cases {
		if got := failureModeString(tc.mode); got != tc.want {
			t.Fatalf("mode %v: got %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestPolicyErrorKind(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{err: controlplane.ErrPolicyNotFound, want: "policy_not_found"},
		{err: controlplane.ErrProviderUnavailable, want: "provider_unavailable"},
		{err: controlplane.ErrPolicyFetchFailed, want: "policy_fetch_failed"},
		{err: errors.New("other"), want: "unknown_error"},
	}

	for _, tc := range cases {
		if got := policyErrorKind(tc.err); got != tc.want {
			t.Fatalf("err=%v: got %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestHandleMissingBudget(t *testing.T) {
	exec := &Executor{missingBudgetMode: FailureAllow}
	if d, ok := exec.handleMissingBudget(context.Background(), "missing"); !ok || !d.Allowed {
		t.Fatalf("expected allow for FailureAllow")
	}

	exec.missingBudgetMode = FailureAllowUnsafe
	if d, ok := exec.handleMissingBudget(context.Background(), "missing"); !ok || !d.Allowed {
		t.Fatalf("expected allow for FailureAllowUnsafe")
	}

	exec.missingBudgetMode = FailureDeny
	if d, ok := exec.handleMissingBudget(context.Background(), "missing"); ok || d.Allowed {
		t.Fatalf("expected deny for FailureDeny")
	}
}

func TestSleepWithContext(t *testing.T) {
	if err := sleepWithContext(context.Background(), 0); err != nil {
		t.Fatalf("expected nil for zero duration, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepWithContext(ctx, 10*time.Millisecond); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
