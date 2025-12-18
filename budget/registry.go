package budget

import (
	"strings"
	"sync"
)

// Registry is a thread-safe name → Budget map.
type Registry struct {
	mu sync.RWMutex
	m  map[string]Budget
}

func NewRegistry() *Registry {
	return &Registry{m: make(map[string]Budget)}
}

// Register associates name with b. Empty names and nil budgets are ignored.
func (r *Registry) Register(name string, b Budget) {
	if r == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" || b == nil {
		return
	}

	r.mu.Lock()
	if r.m == nil {
		r.m = make(map[string]Budget)
	}
	r.m[name] = b
	r.mu.Unlock()
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
