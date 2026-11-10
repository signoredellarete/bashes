package sshtunnel

import (
	"context"
	"errors"
	"sync"
	"testing"

	runtimestate "github.com/signoredellarete/bashes/internal/runtime"
)

func TestManagerStopsTunnelAndClosesResourcesOnce(t *testing.T) {
	manager := NewManager()
	client := &fakeCloser{}
	listener := &fakeCloser{}
	started := make(chan struct{})
	tunnel, err := manager.Register(Options{
		Info:     Info{TunnelID: "tunnel-1", ResourceID: "host-1", Type: "socks"},
		Client:   client,
		Listener: listener,
		Serve: func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	result := make(chan RunResult, 1)
	go func() { result <- manager.Run(tunnel) }()
	<-started

	if !manager.Stop("tunnel-1") {
		t.Fatal("Stop() = false, want true")
	}
	got := <-result
	if !got.Stopped || got.Authoritative {
		t.Fatalf("Run() = %+v, want stopped non-authoritative result", got)
	}
	if client.count() != 1 || listener.count() != 1 {
		t.Fatalf("close counts = client %d, listener %d; want 1 each", client.count(), listener.count())
	}
	if manager.Stop("tunnel-1") {
		t.Fatal("second Stop() = true")
	}
}

func TestManagerReportsNaturalTunnelFailure(t *testing.T) {
	manager := NewManager()
	want := errors.New("listener failed")
	tunnel, err := manager.Register(Options{
		Info:     Info{TunnelID: "tunnel-1", ResourceID: "host-1", Type: "local"},
		Client:   &fakeCloser{},
		Listener: &fakeCloser{},
		Serve:    func(context.Context) error { return want },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	result := manager.Run(tunnel)
	if result.Stopped || !result.Authoritative || !errors.Is(result.Err, want) {
		t.Fatalf("Run() = %+v", result)
	}
	if len(manager.Infos()) != 0 {
		t.Fatalf("Infos() = %v, want no active tunnels", manager.Infos())
	}
}

func TestManagerFailOnlyClosesAuthoritativeTunnel(t *testing.T) {
	manager := NewManager()
	client := &fakeCloser{}
	listener := &fakeCloser{}
	tunnel, err := manager.Register(Options{
		Info:     Info{TunnelID: "tunnel-1", ResourceID: "host-1", Type: "remote"},
		Client:   client,
		Listener: listener,
		Serve:    func(context.Context) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if !manager.Fail(tunnel, errors.New("keepalive failed")) {
		t.Fatal("first Fail() = false")
	}
	if manager.Fail(tunnel, errors.New("keepalive failed")) {
		t.Fatal("second Fail() = true")
	}
	if client.count() != 1 || listener.count() != 1 {
		t.Fatalf("close counts = client %d, listener %d; want 1 each", client.count(), listener.count())
	}
}

func TestManagerListsTunnelIDsByResource(t *testing.T) {
	manager := NewManager()
	for _, info := range []Info{
		{TunnelID: "tunnel-2", ResourceID: "host-1", Type: "socks"},
		{TunnelID: "tunnel-1", ResourceID: "host-2", Type: "local"},
	} {
		if _, err := manager.Register(Options{
			Info: info, Client: &fakeCloser{}, Listener: &fakeCloser{},
			Serve: func(context.Context) error { return nil },
		}); err != nil {
			t.Fatalf("Register(%s) error = %v", info.TunnelID, err)
		}
	}

	if got := manager.IDs(); len(got) != 2 || got[0] != "tunnel-1" || got[1] != "tunnel-2" {
		t.Fatalf("IDs() = %v", got)
	}
	if got := manager.IDsForResource("host-1"); len(got) != 1 || got[0] != "tunnel-2" {
		t.Fatalf("IDsForResource() = %v", got)
	}
}

func TestManagerFailureEmitsFailedLifecycle(t *testing.T) {
	events := make([]runtimestate.Event, 0)
	manager := NewManagerWithObserver(func(event runtimestate.Event) { events = append(events, event) })
	want := errors.New("connection lost")
	tunnel, err := manager.Register(Options{
		Info:   Info{TunnelID: "tunnel-1", ResourceID: "host-1", Type: "socks"},
		Client: &fakeCloser{}, Listener: &fakeCloser{},
		Serve: func(context.Context) error { return want },
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	result := manager.Run(tunnel)
	if !errors.Is(result.Err, want) {
		t.Fatalf("Run() error = %v", result.Err)
	}
	if len(events) != 2 || events[0].Current.State != runtimestate.StateRunning || events[1].Current.State != runtimestate.StateFailed {
		t.Fatalf("lifecycle events = %+v", events)
	}
	if events[1].Current.Error != want.Error() {
		t.Fatalf("failure = %q", events[1].Current.Error)
	}
}

type fakeCloser struct {
	mu     sync.Mutex
	closes int
}

func (closer *fakeCloser) Close() error {
	closer.mu.Lock()
	closer.closes++
	closer.mu.Unlock()
	return nil
}

func (closer *fakeCloser) count() int {
	closer.mu.Lock()
	defer closer.mu.Unlock()
	return closer.closes
}
