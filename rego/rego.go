package rego

import (
	"context"

	"github.com/aponysus/rego/observe"
	"github.com/aponysus/rego/policy"
	"github.com/aponysus/rego/retry"
)

// Key is the structured form of a policy key.
type Key = policy.PolicyKey

// ParseKey parses "namespace.name" into a Key.
func ParseKey(s string) Key { return policy.ParseKey(s) }

// Do executes op using the default executor and the policy for key.
func Do(ctx context.Context, key string, op retry.Operation) error {
	return retry.DefaultExecutor().Do(ctx, policy.ParseKey(key), op)
}

// DoValue executes op using the default executor and the policy for key.
func DoValue[T any](ctx context.Context, key string, op retry.OperationValue[T]) (T, error) {
	return retry.DoValue(ctx, retry.DefaultExecutor(), policy.ParseKey(key), op)
}

// DoWithTimeline executes op using the default executor and returns the Timeline.
func DoWithTimeline(ctx context.Context, key string, op retry.Operation) (observe.Timeline, error) {
	return retry.DefaultExecutor().DoWithTimeline(ctx, policy.ParseKey(key), op)
}

// DoValueWithTimeline executes op using the default executor and returns the Timeline.
func DoValueWithTimeline[T any](ctx context.Context, key string, op retry.OperationValue[T]) (T, observe.Timeline, error) {
	return retry.DoValueWithTimeline(ctx, retry.DefaultExecutor(), policy.ParseKey(key), op)
}
