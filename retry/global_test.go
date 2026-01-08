package retry

import (
	"sync"
	"testing"
)

func resetGlobalExecutor() {
	globalExec = nil
	globalOnce = sync.Once{}
}

func TestDefaultExecutor_LazyInit(t *testing.T) {
	resetGlobalExecutor()

	exec1 := DefaultExecutor()
	if exec1 == nil {
		t.Fatal("expected executor")
	}
	exec2 := DefaultExecutor()
	if exec1 != exec2 {
		t.Fatal("expected DefaultExecutor to return the same instance")
	}
}

func TestSetGlobal_BeforeDefaultExecutor(t *testing.T) {
	resetGlobalExecutor()

	custom := NewExecutor()
	SetGlobal(custom)

	if got := DefaultExecutor(); got != custom {
		t.Fatalf("got %p, want %p", got, custom)
	}
}

func TestSetGlobal_AfterDefaultExecutorIgnored(t *testing.T) {
	resetGlobalExecutor()

	orig := DefaultExecutor()
	custom := NewExecutor()
	SetGlobal(custom)

	if got := DefaultExecutor(); got != orig {
		t.Fatalf("got %p, want %p", got, orig)
	}
}

func TestSetGlobal_IgnoresNil(t *testing.T) {
	resetGlobalExecutor()

	SetGlobal(nil)
	if DefaultExecutor() == nil {
		t.Fatalf("expected default executor to initialize")
	}
}
