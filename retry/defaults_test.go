package retry

import (
	"testing"

	"github.com/aponysus/recourse/classify"
	"github.com/aponysus/recourse/observe"
)

func TestNewDefaultExecutor_ConfiguresDefaults(t *testing.T) {
	exec := NewDefaultExecutor()
	if exec == nil {
		t.Fatal("expected executor")
	}
	if exec.provider == nil || exec.observer == nil || exec.budgets == nil || exec.triggers == nil || exec.classifiers == nil {
		t.Fatal("expected default components to be set")
	}

	if _, ok := exec.classifiers.Get(classify.ClassifierAlwaysRetryOnError); !ok {
		t.Fatalf("expected classifier %q to be registered", classify.ClassifierAlwaysRetryOnError)
	}
	if _, ok := exec.classifiers.Get(classify.ClassifierHTTP); !ok {
		t.Fatalf("expected classifier %q to be registered", classify.ClassifierHTTP)
	}
	if _, ok := exec.classifiers.Get("auto"); !ok {
		t.Fatalf("expected classifier %q to be registered", "auto")
	}

	if _, ok := exec.budgets.Get("unlimited"); !ok {
		t.Fatalf("expected budget %q to be registered", "unlimited")
	}

	if _, ok := exec.triggers.Get("fixed_delay"); !ok {
		t.Fatalf("expected trigger %q to be registered", "fixed_delay")
	}
	if _, ok := exec.triggers.Get("p90"); !ok {
		t.Fatalf("expected trigger %q to be registered", "p90")
	}
	if _, ok := exec.triggers.Get("p95"); !ok {
		t.Fatalf("expected trigger %q to be registered", "p95")
	}
	if _, ok := exec.triggers.Get("p99"); !ok {
		t.Fatalf("expected trigger %q to be registered", "p99")
	}

	if _, ok := exec.observer.(*observe.NoopObserver); !ok {
		t.Fatalf("expected observer to be NoopObserver, got %T", exec.observer)
	}

	if _, ok := exec.defaultClassifier.(classify.AutoClassifier); !ok {
		t.Fatalf("expected default classifier to be AutoClassifier, got %T", exec.defaultClassifier)
	}
}
