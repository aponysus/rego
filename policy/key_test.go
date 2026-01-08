package policy

import "testing"

func TestParseKey_Cases(t *testing.T) {
	cases := []struct {
		input string
		want  PolicyKey
	}{
		{input: "", want: PolicyKey{}},
		{input: "method", want: PolicyKey{Name: "method"}},
		{input: "svc.method", want: PolicyKey{Namespace: "svc", Name: "method"}},
		{input: " svc.method ", want: PolicyKey{Namespace: "svc", Name: "method"}},
		{input: "svc.", want: PolicyKey{Name: "svc."}},
		{input: ".method", want: PolicyKey{Name: "method"}},
		{input: "service . method", want: PolicyKey{Namespace: "service", Name: "method"}},
		{input: "svc.method.extra", want: PolicyKey{Namespace: "svc", Name: "method.extra"}},
	}

	for _, tc := range cases {
		if got := ParseKey(tc.input); got != tc.want {
			t.Fatalf("ParseKey(%q) = %+v, want %+v", tc.input, got, tc.want)
		}
	}
}

func TestPolicyKey_String(t *testing.T) {
	cases := []struct {
		key  PolicyKey
		want string
	}{
		{key: PolicyKey{}, want: ""},
		{key: PolicyKey{Name: "method"}, want: "method"},
		{key: PolicyKey{Namespace: "svc"}, want: "svc"},
		{key: PolicyKey{Namespace: "svc", Name: "method"}, want: "svc.method"},
	}

	for _, tc := range cases {
		if got := tc.key.String(); got != tc.want {
			t.Fatalf("String(%+v) = %q, want %q", tc.key, got, tc.want)
		}
	}
}
