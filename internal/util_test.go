package internal

import "testing"

func TestIsTypedNil(t *testing.T) {
	var (
		nilPtr   *int
		nilSlice []string
		nilMap   map[string]int
		nilFunc  func()
		nilChan  chan int
	)

	cases := []struct {
		name string
		val  any
		want bool
	}{
		{name: "nil", val: nil, want: true},
		{name: "nil_ptr", val: nilPtr, want: true},
		{name: "nil_slice", val: nilSlice, want: true},
		{name: "nil_map", val: nilMap, want: true},
		{name: "nil_func", val: nilFunc, want: true},
		{name: "nil_chan", val: nilChan, want: true},
		{name: "typed_nil_interface", val: any(nilPtr), want: true},
		{name: "non_nil_ptr", val: new(int), want: false},
		{name: "non_nil_value", val: 123, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTypedNil(tc.val); got != tc.want {
				t.Fatalf("IsTypedNil(%v)=%v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
