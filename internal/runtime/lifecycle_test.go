package runtime

import (
	"errors"
	"testing"
	"time"
)

func TestLifecycleReportsOrderedTransitions(t *testing.T) {
	now := time.Unix(100, 0)
	events := make([]Event, 0)
	lifecycle, err := NewLifecycle(Options{
		ID: "session-1", ResourceID: "host-1", Kind: "terminal",
		Now: func() time.Time {
			now = now.Add(time.Second)
			return now
		},
		Observer: func(event Event) { events = append(events, event) },
	})
	if err != nil {
		t.Fatalf("NewLifecycle() error = %v", err)
	}
	if !lifecycle.MarkRunning() || !lifecycle.RequestStop() || !lifecycle.Complete(nil) {
		t.Fatal("valid lifecycle transition was rejected")
	}
	if lifecycle.Complete(nil) {
		t.Fatal("terminal state accepted another transition")
	}
	if got := lifecycle.Snapshot(); got.State != StateStopped || got.ID != "session-1" || got.Error != "" {
		t.Fatalf("Snapshot() = %+v", got)
	}
	if len(events) != 3 || events[0].Previous != StateStarting || events[2].Current.State != StateStopped {
		t.Fatalf("events = %+v", events)
	}
}

func TestLifecycleRecordsFailure(t *testing.T) {
	lifecycle, err := NewLifecycle(Options{ID: "tunnel-1", ResourceID: "host-1", Kind: "tunnel"})
	if err != nil {
		t.Fatalf("NewLifecycle() error = %v", err)
	}
	lifecycle.MarkRunning()
	lifecycle.Complete(errors.New("connection lost"))
	if got := lifecycle.Snapshot(); got.State != StateFailed || got.Error != "connection lost" {
		t.Fatalf("Snapshot() = %+v", got)
	}
}

func TestLifecycleValidatesIdentity(t *testing.T) {
	if _, err := NewLifecycle(Options{ID: "runtime-1", Kind: "terminal"}); err == nil {
		t.Fatal("NewLifecycle() accepted an empty resource id")
	}
}
