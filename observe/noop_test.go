package observe_test

import (
	"context"
	"testing"

	"github.com/aponysus/recourse/observe"
	"github.com/aponysus/recourse/policy"
)

func TestNoopObserver_HandlesEvents(t *testing.T) {
	obs := observe.NoopObserver{}
	ctx := context.Background()
	key := policy.PolicyKey{Name: "op"}
	pol := policy.DefaultPolicyFor(key)
	rec := observe.AttemptRecord{Attempt: 1}
	ev := observe.BudgetDecisionEvent{Key: key, Attempt: 1}
	tl := observe.Timeline{Key: key}

	obs.OnStart(ctx, key, pol)
	obs.OnAttempt(ctx, key, rec)
	obs.OnHedgeSpawn(ctx, key, rec)
	obs.OnHedgeCancel(ctx, key, rec, "reason")
	obs.OnBudgetDecision(ctx, ev)
	obs.OnSuccess(ctx, key, tl)
	obs.OnFailure(ctx, key, tl)
}

func TestBaseObserver_HandlesEvents(t *testing.T) {
	obs := observe.BaseObserver{}
	ctx := context.Background()
	key := policy.PolicyKey{Name: "op"}
	pol := policy.DefaultPolicyFor(key)
	rec := observe.AttemptRecord{Attempt: 1}
	ev := observe.BudgetDecisionEvent{Key: key, Attempt: 1}
	tl := observe.Timeline{Key: key}

	obs.OnStart(ctx, key, pol)
	obs.OnAttempt(ctx, key, rec)
	obs.OnHedgeSpawn(ctx, key, rec)
	obs.OnHedgeCancel(ctx, key, rec, "reason")
	obs.OnBudgetDecision(ctx, ev)
	obs.OnSuccess(ctx, key, tl)
	obs.OnFailure(ctx, key, tl)
}
