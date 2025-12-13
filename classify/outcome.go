package classify

import "time"

// OutcomeKind describes the executor's decision about an attempt result.
type OutcomeKind int

const (
	OutcomeUnknown OutcomeKind = iota
	OutcomeSuccess
	OutcomeRetryable
	OutcomeNonRetryable
	OutcomeAbort
)

// Outcome describes the classification of an attempt.
type Outcome struct {
	Kind   OutcomeKind
	Reason string

	// BackoffOverride, when set, overrides the policy backoff before the next attempt.
	BackoffOverride time.Duration
}
