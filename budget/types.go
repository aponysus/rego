package budget

import (
	"context"

	"github.com/aponysus/recourse/policy"
)

// AttemptKind describes the attempt type being gated.
type AttemptKind int

const (
	KindRetry AttemptKind = iota
	KindHedge
)

// Standard Decision.Reason strings.
const (
	ReasonNoBudget       = "no_budget"
	ReasonBudgetNotFound = "budget_not_found"
	ReasonBudgetDenied   = "budget_denied"
	ReasonPanicInBudget  = "panic_in_budget"
)

// Decision is the result of a budget check.
type Decision struct {
	Allowed bool
	Reason  string

	// Release, when non-nil, is called exactly once after an allowed attempt finishes.
	Release func()
}

// Budget gates attempts to prevent retry/hedge storms.
type Budget interface {
	AllowAttempt(ctx context.Context, key policy.PolicyKey, attemptIdx int, kind AttemptKind, ref policy.BudgetRef) Decision
}
