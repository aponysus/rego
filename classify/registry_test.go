package classify

import "testing"

type testClassifier struct{}

func (testClassifier) Classify(any, error) Outcome {
	return Outcome{Kind: OutcomeSuccess, Reason: "ok"}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("expected registry")
	}

	reg.Register("  custom  ", testClassifier{})
	got, ok := reg.Get("custom")
	if !ok || got == nil {
		t.Fatal("expected classifier to be registered")
	}
}

func TestRegistry_Validation(t *testing.T) {
	var nilReg *Registry
	nilReg.Register("name", testClassifier{})
	if got, ok := nilReg.Get("name"); ok || got != nil {
		t.Fatalf("expected nil,false for nil registry")
	}

	reg := NewRegistry()
	reg.Register("   ", testClassifier{})
	if got, ok := reg.Get("   "); ok || got != nil {
		t.Fatalf("expected empty name to be ignored")
	}

	reg.Register("name", nil)
	if got, ok := reg.Get("name"); ok || got != nil {
		t.Fatalf("expected nil classifier to be ignored")
	}
}
