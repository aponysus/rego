package budget

import (
	"context"
	"testing"

	"github.com/aponysus/recourse/policy"
)

type testBudget struct{}

func (testBudget) AllowAttempt(context.Context, policy.PolicyKey, int, AttemptKind, policy.BudgetRef) Decision {
	return Decision{Allowed: true, Reason: ReasonAllowed}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("expected registry")
	}

	if err := reg.Register(" primary ", testBudget{}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	b, ok := reg.Get("primary")
	if !ok || b == nil {
		t.Fatalf("expected budget to be registered")
	}
}

func TestRegistry_RegisterValidation(t *testing.T) {
	var nilReg *Registry
	if err := nilReg.Register("x", testBudget{}); err == nil {
		t.Fatal("expected error for nil registry")
	}

	reg := NewRegistry()
	if err := reg.Register("   ", testBudget{}); err == nil {
		t.Fatal("expected error for empty name")
	}

	var nilBudget *testBudget
	if err := reg.Register("x", nilBudget); err == nil {
		t.Fatal("expected error for typed-nil budget")
	}
}

func TestRegistry_MustRegisterPanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	reg := NewRegistry()
	reg.MustRegister("", testBudget{})
}

func TestRegistry_GetValidation(t *testing.T) {
	var nilReg *Registry
	if b, ok := nilReg.Get("x"); b != nil || ok {
		t.Fatalf("expected nil,false for nil registry")
	}

	reg := NewRegistry()
	if b, ok := reg.Get(" "); b != nil || ok {
		t.Fatalf("expected nil,false for empty name")
	}
}
