package budget

import (
	"errors"
	"strings"
	"sync"

	"github.com/aponysus/recourse/internal"
)

// Registry is a thread-safe name → Budget map.
type Registry struct {
	mu sync.RWMutex
	m  map[string]Budget
}

func NewRegistry() *Registry {
	return &Registry{m: make(map[string]Budget)}
}

// Register registers a budget with validation.
// It returns an error if the registry is nil, the name is empty, or the budget is nil/typed-nil.
func (r *Registry) Register(name string, b Budget) error {
	if r == nil {
		return errors.New("registry is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("budget name cannot be empty")
	}
	if internal.IsTypedNil(b) {
		return errors.New("budget cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.m == nil {
		r.m = make(map[string]Budget)
	}
	r.m[name] = b
	return nil
}

// MustRegister registers a budget and panics on error.
func (r *Registry) MustRegister(name string, b Budget) {
	if err := r.Register(name, b); err != nil {
		panic("budget.Registry.MustRegister: " + err.Error())
	}
}

func (r *Registry) Get(name string) (Budget, bool) {
	if r == nil {
		return nil, false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false
	}

	r.mu.RLock()
	b, ok := r.m[name]
	r.mu.RUnlock()
	return b, ok && b != nil
}
