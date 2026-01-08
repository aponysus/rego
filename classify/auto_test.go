package classify

import (
	"errors"
	"testing"
	"time"
)

type httpErr struct {
	status int
	method string
}

func (e httpErr) Error() string                     { return "http err" }
func (e httpErr) HTTPStatusCode() int               { return e.status }
func (e httpErr) HTTPMethod() string                { return e.method }
func (e httpErr) RetryAfter() (time.Duration, bool) { return 0, false }

func TestAutoClassifier_HTTPError(t *testing.T) {
	out := AutoClassifier{}.Classify(nil, httpErr{status: 500, method: "GET"})
	if out.Kind != OutcomeRetryable || out.Reason != "http_5xx" {
		t.Fatalf("out=%+v, want retryable http_5xx", out)
	}
}

func TestAutoClassifier_NonHTTPError(t *testing.T) {
	out := AutoClassifier{}.Classify(nil, errors.New("boom"))
	if out.Kind != OutcomeRetryable || out.Reason != "retryable_error" {
		t.Fatalf("out=%+v, want retryable_error", out)
	}
}
