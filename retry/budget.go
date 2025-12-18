package retry

import (
	"context"
	"strings"

	"github.com/aponysus/rego/budget"
	"github.com/aponysus/rego/policy"
)

func (e *Executor) allowAttempt(ctx context.Context, key policy.PolicyKey, ref policy.BudgetRef, attemptIdx int, kind budget.AttemptKind) (decision budget.Decision, allowed bool) {
	if e == nil {
		return budget.Decision{Allowed: true, Reason: budget.ReasonNoBudget}, true
	}

	ref.Name = strings.TrimSpace(ref.Name)
	if ref.Name == "" || e.budgets == nil {
		return budget.Decision{Allowed: true, Reason: budget.ReasonNoBudget}, true
	}

	b, ok := e.budgets.Get(ref.Name)
	if !ok {
		allowed = e.missingBudgetMode != FailureDeny
		return budget.Decision{Allowed: allowed, Reason: budget.ReasonBudgetNotFound}, allowed
	}

	if e.recoverPanics {
		defer func() {
			if r := recover(); r != nil {
				decision = budget.Decision{Allowed: false, Reason: budget.ReasonPanicInBudget}
				allowed = false
			}
		}()
	}

	decision = b.AllowAttempt(ctx, key, attemptIdx, kind, ref)
	if decision.Reason == "" {
		if decision.Allowed {
			decision.Reason = "allowed"
		} else {
			decision.Reason = budget.ReasonBudgetDenied
		}
	}
	return decision, decision.Allowed
}
