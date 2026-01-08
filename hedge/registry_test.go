package hedge

import (
	"testing"
	"time"
)

type testTrigger struct{}

func (testTrigger) ShouldSpawnHedge(HedgeState) (bool, time.Duration) { return false, 0 }

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register("fixed", testTrigger{})

	got, ok := reg.Get("fixed")
	if !ok || got == nil {
		t.Fatal("expected trigger to be registered")
	}
}

func TestRegistry_RegisterPanics(t *testing.T) {
	reg := NewRegistry()

	expectPanic := func(fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()
		fn()
	}

	expectPanic(func() { reg.Register("", testTrigger{}) })
	expectPanic(func() { reg.Register("x", nil) })
}
