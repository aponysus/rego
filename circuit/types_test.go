package circuit

import "testing"

func TestStateString(t *testing.T) {
	cases := []struct {
		state State
		want  string
	}{
		{state: StateClosed, want: "closed"},
		{state: StateOpen, want: "open"},
		{state: StateHalfOpen, want: "half-open"},
		{state: State(99), want: "unknown"},
	}

	for _, tc := range cases {
		if got := tc.state.String(); got != tc.want {
			t.Fatalf("state %v: got %q, want %q", tc.state, got, tc.want)
		}
	}
}
