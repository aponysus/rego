package retry

import "sync"

var (
	defaultOnce sync.Once
	defaultExec *Executor
)

// DefaultExecutor returns a lazily-initialized executor configured with
// conservative defaults.
func DefaultExecutor() *Executor {
	defaultOnce.Do(func() {
		defaultExec = NewExecutor(ExecutorOptions{})
	})
	return defaultExec
}
