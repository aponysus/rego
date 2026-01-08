package policy

import "testing"

func TestNormalizeError_Error(t *testing.T) {
	var err *NormalizeError
	if got := err.Error(); got != "<nil>" {
		t.Fatalf("nil error string=%q, want %q", got, "<nil>")
	}

	err = &NormalizeError{Field: "retry.jitter", Value: "bogus"}
	got := err.Error()
	if got == "" || got == "<nil>" {
		t.Fatalf("unexpected error string: %q", got)
	}
}
