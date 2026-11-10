package transfer

import (
	"testing"

	runtimestate "github.com/signoredellarete/bashes/internal/runtime"
)

func TestSessionManagerLifecycleAndResourceLookup(t *testing.T) {
	manager := NewSessionManager[*testSession]()
	first := &testSession{}
	second := &testSession{}
	register := func(id string, resourceID string, session *testSession) {
		t.Helper()
		if err := manager.Register(SessionOptions[*testSession]{
			ID: id, ResourceID: resourceID, Value: session,
			Close: func(value *testSession) { value.closes++ },
		}); err != nil {
			t.Fatalf("Register(%s) error = %v", id, err)
		}
	}
	register("files-2", "host-1", second)
	register("files-1", "host-2", first)

	if got := manager.IDs(); len(got) != 2 || got[0] != "files-1" || got[1] != "files-2" {
		t.Fatalf("IDs() = %v", got)
	}
	if got := manager.IDsForResource("host-1"); len(got) != 1 || got[0] != "files-2" {
		t.Fatalf("IDsForResource() = %v", got)
	}
	if got, ok := manager.Get("files-1"); !ok || got != first {
		t.Fatalf("Get() = %v, %v", got, ok)
	}
	if !manager.Stop("files-1") || first.closes != 1 {
		t.Fatalf("Stop() did not close session once: %+v", first)
	}
	if manager.Stop("files-1") || first.closes != 1 {
		t.Fatalf("second Stop() changed close count: %+v", first)
	}
}

func TestSessionManagerRejectsDuplicateAndInvalidSessions(t *testing.T) {
	manager := NewSessionManager[int]()
	options := SessionOptions[int]{ID: "files-1", ResourceID: "host-1", Value: 1, Close: func(int) {}}
	if err := manager.Register(options); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := manager.Register(options); err == nil {
		t.Fatal("duplicate Register() error = nil")
	}
	if err := manager.Register(SessionOptions[int]{ResourceID: "host-1", Close: func(int) {}}); err == nil {
		t.Fatal("invalid Register() error = nil")
	}
}

func TestSessionManagerEmitsLifecycle(t *testing.T) {
	events := make([]runtimestate.Event, 0)
	manager := NewSessionManagerWithObserver[int](func(event runtimestate.Event) { events = append(events, event) })
	if err := manager.Register(SessionOptions[int]{
		ID: "files-1", ResourceID: "host-1", Value: 1, Close: func(int) {},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got := manager.Snapshots(); len(got) != 1 || got[0].State != runtimestate.StateRunning {
		t.Fatalf("Snapshots() = %+v", got)
	}
	manager.Stop("files-1")
	if len(events) != 3 || events[2].Current.State != runtimestate.StateStopped {
		t.Fatalf("lifecycle events = %+v", events)
	}
}

type testSession struct {
	closes int
}
