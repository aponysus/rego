package hedge

import (
	"testing"
	"time"
)

func TestFixedDelayTrigger_UsesDelayAndElapsed(t *testing.T) {
	trigger := FixedDelayTrigger{Delay: 100 * time.Millisecond}

	state := HedgeState{
		AttemptsLaunched: 1,
		MaxHedges:        1,
		Elapsed:          50 * time.Millisecond,
	}

	spawn, wait := trigger.ShouldSpawnHedge(state)
	if spawn || wait != 50*time.Millisecond {
		t.Fatalf("spawn=%v wait=%v, want false/50ms", spawn, wait)
	}

	state.Elapsed = 100 * time.Millisecond
	spawn, wait = trigger.ShouldSpawnHedge(state)
	if !spawn || wait != 0 {
		t.Fatalf("spawn=%v wait=%v, want true/0", spawn, wait)
	}
}

func TestFixedDelayTrigger_RespectsOverrideAndMaxAttempts(t *testing.T) {
	trigger := FixedDelayTrigger{Delay: 100 * time.Millisecond}

	state := HedgeState{
		AttemptsLaunched: 1,
		MaxHedges:        2,
		Elapsed:          10 * time.Millisecond,
		HedgeDelay:       40 * time.Millisecond,
	}

	spawn, wait := trigger.ShouldSpawnHedge(state)
	if spawn || wait != 30*time.Millisecond {
		t.Fatalf("spawn=%v wait=%v, want false/30ms", spawn, wait)
	}

	state.AttemptsLaunched = 3 // primary + max hedges
	spawn, wait = trigger.ShouldSpawnHedge(state)
	if spawn || wait != 0 {
		t.Fatalf("spawn=%v wait=%v, want false/0", spawn, wait)
	}
}
