package retry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aponysus/recourse/classify"
	"github.com/aponysus/recourse/policy"
)

type fuzzClassifier struct{}

func (fuzzClassifier) Classify(_ any, err error) classify.Outcome {
	if err == nil {
		return classify.Outcome{Kind: classify.OutcomeSuccess, Reason: "success"}
	}
	if errors.Is(err, context.Canceled) {
		return classify.Outcome{Kind: classify.OutcomeAbort, Reason: "context_canceled"}
	}
	if strings.HasPrefix(err.Error(), "nonretry") {
		return classify.Outcome{Kind: classify.OutcomeNonRetryable, Reason: "non_retryable"}
	}
	return classify.Outcome{Kind: classify.OutcomeRetryable, Reason: "retryable_error"}
}

func FuzzExecutor(f *testing.F) {
	f.Add(3, "retry-policy")
	f.Add(0, "no-retry")
	f.Add(-2, "negative")
	f.Add(7, "nonretry")

	f.Fuzz(func(t *testing.T, attempts int, polName string) {
		if attempts < -50 || attempts > 50 {
			return
		}
		if len(polName) > 64 {
			return
		}
		key := policy.PolicyKey{Name: polName}
		normalized := policy.NewFromKey(
			key,
			policy.MaxAttempts(attempts),
			policy.InitialBackoff(0),
			policy.MaxBackoff(0),
			policy.Jitter(policy.JitterNone),
		)
		maxAttempts := normalized.Retry.MaxAttempts
		if maxAttempts < 1 {
			maxAttempts = 1
		}

		mode := attempts % 3
		if mode < 0 {
			mode = -mode
		}
		failFor := 0
		if maxAttempts > 0 {
			failFor = abs(attempts) % (maxAttempts + 1)
		}

		exec := NewExecutor(
			WithPolicyKey(
				key,
				policy.MaxAttempts(attempts),
				policy.InitialBackoff(0),
				policy.MaxBackoff(0),
				policy.Jitter(policy.JitterNone),
			),
			WithDefaultClassifier(fuzzClassifier{}),
		)
		exec.sleep = func(context.Context, time.Duration) error { return nil }

		calls := 0
		_, err := DoValue[int](context.Background(), exec, key, func(ctx context.Context) (int, error) {
			calls++
			if calls <= failFor {
				switch mode {
				case 0:
					return 0, errors.New("nonretryable_failure")
				case 1:
					return 0, context.Canceled
				default:
					return 0, errors.New("retryable_failure")
				}
			}
			return 1, nil
		})
		if calls > maxAttempts {
			t.Fatalf("calls=%d exceeded maxAttempts=%d", calls, maxAttempts)
		}
		if failFor == 0 && err != nil {
			t.Fatalf("unexpected error on immediate success: %v", err)
		}
		if mode == 0 && failFor > 0 && calls != 1 {
			t.Fatalf("nonretryable should stop after 1 call, got %d", calls)
		}
		if mode == 1 && failFor > 0 && calls != 1 {
			t.Fatalf("canceled should stop after 1 call, got %d", calls)
		}
		if mode == 2 && failFor < maxAttempts && err != nil {
			t.Fatalf("unexpected error for retryable path: %v", err)
		}
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
