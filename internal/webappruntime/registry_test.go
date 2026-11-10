package webappruntime

import (
	"testing"

	runtimestate "github.com/signoredellarete/bashes/internal/runtime"
)

func TestRegistrySharesRuntimeAcrossApps(t *testing.T) {
	registry := NewRegistry[*testRuntime]()
	runtime := &testRuntime{id: "runtime-1"}
	if err := registry.Register(Options[*testRuntime]{
		RuntimeID: runtime.id, GroupKey: "firefox:profile", AppID: "app-1", Value: runtime,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := registry.Attach(runtime.id, "app-2"); err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if got, ok := registry.FindByGroup("firefox:profile"); !ok || got != runtime {
		t.Fatalf("FindByGroup() = %v, %v", got, ok)
	}
	if got := registry.AppIDs(runtime.id); len(got) != 2 || got[0] != "app-1" || got[1] != "app-2" {
		t.Fatalf("AppIDs() = %v", got)
	}
	if got := registry.Bindings(); len(got) != 2 || got[0].AppID != "app-1" || got[1].AppID != "app-2" {
		t.Fatalf("Bindings() = %v", got)
	}
}

func TestRegistryDetachAndRemoveAreAuthoritative(t *testing.T) {
	registry := NewRegistry[*testRuntime]()
	runtime := &testRuntime{id: "runtime-1"}
	if err := registry.Register(Options[*testRuntime]{
		RuntimeID: runtime.id, GroupKey: "group-1", AppID: "app-1", Value: runtime,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	got, remaining, found := registry.DetachApp("app-1")
	if !found || got != runtime || remaining != 0 {
		t.Fatalf("DetachApp() = %v, %d, %v", got, remaining, found)
	}
	if registry.Remove(runtime.id, &testRuntime{id: runtime.id}) {
		t.Fatal("Remove() accepted a stale runtime instance")
	}
	if !registry.Remove(runtime.id, runtime) {
		t.Fatal("Remove() = false")
	}
	if _, ok := registry.Get(runtime.id); ok {
		t.Fatal("removed runtime is still registered")
	}
}

func TestRegistryReassignsAppWithoutLeavingStaleBinding(t *testing.T) {
	registry := NewRegistry[*testRuntime]()
	first := &testRuntime{id: "runtime-1"}
	second := &testRuntime{id: "runtime-2"}
	for _, options := range []Options[*testRuntime]{
		{RuntimeID: first.id, GroupKey: "group-1", AppID: "app-1", Value: first},
		{RuntimeID: second.id, GroupKey: "group-2", AppID: "app-2", Value: second},
	} {
		if err := registry.Register(options); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}
	if err := registry.Attach(second.id, "app-1"); err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if got, ok := registry.ForApp("app-1"); !ok || got != second {
		t.Fatalf("ForApp() = %v, %v", got, ok)
	}
	if got := registry.AppIDs(first.id); len(got) != 0 {
		t.Fatalf("first runtime apps = %v", got)
	}
}

func TestRegistryEmitsLifecycle(t *testing.T) {
	events := make([]runtimestate.Event, 0)
	registry := NewRegistryWithObserver[*testRuntime](func(event runtimestate.Event) { events = append(events, event) })
	runtime := &testRuntime{id: "runtime-1"}
	if err := registry.Register(Options[*testRuntime]{
		RuntimeID: runtime.id, GroupKey: "group-1", AppID: "app-1", Value: runtime,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got := registry.Snapshots(); len(got) != 1 || got[0].State != runtimestate.StateRunning {
		t.Fatalf("Snapshots() = %+v", got)
	}
	if !registry.Remove(runtime.id, runtime) {
		t.Fatal("Remove() = false")
	}
	if len(events) != 3 || events[2].Current.State != runtimestate.StateStopped {
		t.Fatalf("lifecycle events = %+v", events)
	}
}

func TestRegistryReportsRuntimeFailure(t *testing.T) {
	events := make([]runtimestate.Event, 0)
	registry := NewRegistryWithObserver[*testRuntime](func(event runtimestate.Event) { events = append(events, event) })
	runtime := &testRuntime{id: "runtime-1"}
	if err := registry.Register(Options[*testRuntime]{
		RuntimeID: runtime.id, GroupKey: "group-1", AppID: "app-1", Value: runtime,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !registry.Fail(runtime.id, runtime, errTestRuntimeFailure) {
		t.Fatal("Fail() = false")
	}
	if len(events) != 2 || events[1].Current.State != runtimestate.StateFailed || events[1].Current.Error != errTestRuntimeFailure.Error() {
		t.Fatalf("lifecycle events = %+v", events)
	}
}

var errTestRuntimeFailure = &testRuntimeError{}

type testRuntimeError struct{}

func (*testRuntimeError) Error() string { return "browser crashed" }

type testRuntime struct {
	id string
}
